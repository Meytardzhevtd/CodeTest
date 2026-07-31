package worker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const (
	// Лимиты пока захардкожены: per-task time_limit_ms/memory_limit_mb лежат
	// в Postgres, а координатор к ней не подключён и в judgepb.Task эти поля
	// не передаются. Когда это появится, testTimeLimit/memoryLimitBytes нужно
	// будет брать из task, а не из констант.
	testTimeLimit = 5 * time.Second
	// Компиляция Go в контейнере без прогретого GOCACHE собирает с нуля ещё и
	// стандартную библиотеку — таймаут взят с запасом на этот холодный старт
	// (см. cacheVolume у "go" в languageSpecs, который делает все последующие
	// сборки быстрыми за счёт переиспользования кэша между контейнерами).
	compileTimeLimit = 40 * time.Second
	memoryLimitMiB   = 256
	execPidsLimit    = 64
	execCPULimit     = 1_000_000_000  // 1 vCPU, в единицах NanoCPUs
	dockerCallSlack  = 2 * time.Second // запас поверх лимита на накладные расходы Docker API
)

// languageSpec описывает всё, что нужно воркеру, чтобы собрать (если нужно)
// и запустить решение на конкретном языке внутри контейнера с этим образом.
type languageSpec struct {
	image      string
	sourceFile string // имя файла с решением внутри контейнера, кладётся в /tmp
	compileCmd string // пусто — компиляция не нужна (интерпретируемый язык)
	runCmd     string

	// cacheVolume — именованный Docker-volume, монтируемый в cacheMountPath.
	// Каждый сабмишн получает свежий контейнер, так что без persistent-тома
	// компилятору неоткуда взять кэш и каждый раз он собирает всё с нуля.
	// Для Go это особенно чувствительно: "go build" без прогретого GOCACHE
	// компилирует ещё и стандартную библиотеку, что может занимать дольше
	// lease-таймаута координатора.
	cacheVolume    string
	cacheMountPath string
}

var languageSpecs = map[string]languageSpec{
	"python": {
		image:      "python:3.12-slim",
		sourceFile: "solution.py",
		runCmd:     "python3 /tmp/solution.py",
	},
	"cpp": {
		image:      "gcc:13",
		sourceFile: "solution.cpp",
		compileCmd: "g++ -O2 -o /tmp/solution /tmp/solution.cpp",
		runCmd:     "/tmp/solution",
	},
	"go": {
		image:          "golang:1.23",
		sourceFile:     "solution.go",
		compileCmd:     "go build -o /tmp/solution /tmp/solution.go",
		runCmd:         "/tmp/solution",
		cacheVolume:    "codetest-go-build-cache",
		cacheMountPath: "/root/.cache/go-build",
	},
}

// DockerJudge исполняет сабмишны в изолированных контейнерах через Docker
// Engine API. Кеширования тест-кейсов тут нет и не должно быть — решение об
// этом принимается координатором, воркер просто получает готовый
// task.TestCases и гоняет его.
type DockerJudge struct {
	cli *client.Client
}

func NewDockerJudge() (*DockerJudge, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &DockerJudge{cli: cli}, nil
}

// EnsureImage подтягивает образы всех поддерживаемых языков, если их ещё нет
// в демоне, к которому подключён воркер (отдельный DinD-демон не шарит кэш
// образов с хостом).
func (d *DockerJudge) EnsureImage(ctx context.Context) error {
	pulled := make(map[string]bool, len(languageSpecs))
	for _, spec := range languageSpecs {
		if pulled[spec.image] {
			continue
		}
		pulled[spec.image] = true

		rc, err := d.cli.ImagePull(ctx, spec.image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("pull image %s: %w", spec.image, err)
		}
		_, copyErr := io.Copy(io.Discard, rc)
		rc.Close()
		if copyErr != nil {
			return fmt.Errorf("pull image %s: %w", spec.image, copyErr)
		}
	}
	return nil
}

// Judge поднимает один контейнер на весь сабмишн, при необходимости собирает
// решение и прогоняет каждый тест через отдельный exec внутри него.
// Останавливается на первом же непрошедшем тесте (типичное поведение
// чекера) — остальные не запускаются.
func (d *DockerJudge) Judge(ctx context.Context, task *judgepb.Task) *judgepb.SubmitResultRequest {
	spec, ok := languageSpecs[task.Language]
	if !ok {
		return errorResult(task.SubmissionId, fmt.Sprintf("неподдерживаемый язык: %q", task.Language))
	}

	containerID, err := d.createContainer(ctx, spec)
	if err != nil {
		return errorResult(task.SubmissionId, fmt.Sprintf("создание песочницы: %v", err))
	}
	defer func() {
		removeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.cli.ContainerRemove(removeCtx, containerID, container.RemoveOptions{Force: true})
	}()

	if err := d.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return errorResult(task.SubmissionId, fmt.Sprintf("запуск песочницы: %v", err))
	}

	if err := d.copyCode(ctx, containerID, spec.sourceFile, task.Code); err != nil {
		return errorResult(task.SubmissionId, fmt.Sprintf("копирование решения: %v", err))
	}

	if spec.compileCmd != "" {
		exitCode, _, stderr, err := d.exec(ctx, containerID, spec.compileCmd, "", compileTimeLimit)
		if err != nil {
			return errorResult(task.SubmissionId, fmt.Sprintf("компиляция: %v", err))
		}
		if exitCode != 0 {
			return &judgepb.SubmitResultRequest{
				SubmissionId: task.SubmissionId,
				Status:       judgepb.Status_STATUS_CE,
				Error:        stderr,
			}
		}
	}

	for _, tc := range task.TestCases {
		status, output, stderr, err := d.runTest(ctx, containerID, spec.runCmd, tc)
		if err != nil {
			return errorResult(task.SubmissionId, fmt.Sprintf("тест %03d: %v", tc.Number, err))
		}
		if status != judgepb.Status_STATUS_OK {
			return &judgepb.SubmitResultRequest{
				SubmissionId: task.SubmissionId,
				Status:       status,
				Output:       output,
				Error:        stderr,
			}
		}
	}

	return &judgepb.SubmitResultRequest{
		SubmissionId: task.SubmissionId,
		Status:       judgepb.Status_STATUS_OK,
	}
}

func (d *DockerJudge) createContainer(ctx context.Context, spec languageSpec) (string, error) {
	pidsLimit := int64(execPidsLimit)
	memoryLimitBytes := int64(memoryLimitMiB) * 1024 * 1024

	var mounts []mount.Mount
	if spec.cacheVolume != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: spec.cacheVolume,
			Target: spec.cacheMountPath,
		})
	}

	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image: spec.image,
			Cmd:   []string{"sleep", "infinity"},
		},
		&container.HostConfig{
			NetworkMode: "none",
			Mounts:      mounts,
			Resources: container.Resources{
				Memory:     memoryLimitBytes,
				MemorySwap: memoryLimitBytes, // без своп-надбавки, иначе лимит по факту в 2 раза больше
				NanoCPUs:   execCPULimit,
				PidsLimit:  &pidsLimit,
			},
		},
		nil, nil, "",
	)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (d *DockerJudge) copyCode(ctx context.Context, containerID, fileName, code string) error {
	body := []byte(code)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: fileName,
		Mode: 0o644,
		Size: int64(len(body)),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(body); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	return d.cli.CopyToContainer(ctx, containerID, "/tmp", &buf, container.CopyToContainerOptions{})
}

// runTest запускает уже собранное (для компилируемых языков) или интерпретируемое
// решение в отдельном exec поверх уже поднятого контейнера и переводит код
// возврата в вердикт. Реальный лимит по времени навязывается изнутри
// контейнера через `timeout` (код возврата 124), а не через context — это
// надёжнее, чем пытаться убить конкретный exec-процесс снаружи через Docker
// API, для которого нет прямого "kill exec" эндпоинта.
func (d *DockerJudge) runTest(ctx context.Context, containerID, runCmd string, tc *judgepb.TestCase) (judgepb.Status, string, string, error) {
	cmd := fmt.Sprintf("timeout %gs %s", testTimeLimit.Seconds(), runCmd)
	exitCode, stdout, stderr, err := d.exec(ctx, containerID, cmd, tc.Input, testTimeLimit+dockerCallSlack)
	if err != nil {
		return 0, "", "", err
	}

	switch exitCode {
	case 0:
		if normalizeOutput(stdout) == normalizeOutput(tc.ExpectedOutput) {
			return judgepb.Status_STATUS_OK, "", "", nil
		}
		return judgepb.Status_STATUS_WA, stdout, "", nil
	case 124:
		// код возврата coreutils timeout при принудительном убийстве по таймауту
		return judgepb.Status_STATUS_TL, "", "", nil
	case 137:
		// SIGKILL — в т.ч. OOM killer при превышении лимита памяти контейнера;
		// это эвристика, точнее без чтения cgroup-событий не отличить от
		// любого другого SIGKILL, но для MVP этого достаточно
		return judgepb.Status_STATUS_ML, "", "", nil
	default:
		return judgepb.Status_STATUS_RE, stdout, stderr, nil
	}
}

// exec запускает команду в уже поднятом контейнере, передаёт stdin (если
// есть) и возвращает код возврата и вывод. Используется и для компиляции, и
// для прогона тестов — единственная разница между ними в том, что подаётся
// на stdin и как трактуется код возврата.
func (d *DockerJudge) exec(ctx context.Context, containerID, cmd, stdin string, timeout time.Duration) (int, string, string, error) {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	execResp, err := d.cli.ContainerExecCreate(execCtx, containerID, container.ExecOptions{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("exec create: %w", err)
	}

	hijacked, err := d.cli.ContainerExecAttach(execCtx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return 0, "", "", fmt.Errorf("exec attach: %w", err)
	}
	defer hijacked.Close()

	if _, err := hijacked.Conn.Write([]byte(stdin)); err != nil {
		return 0, "", "", fmt.Errorf("запись stdin: %w", err)
	}

	cw, ok := hijacked.Conn.(interface{ CloseWrite() error })
	if !ok {
		return 0, "", "", fmt.Errorf("соединение с Docker-демоном не поддерживает half-close stdin")
	}
	if err := cw.CloseWrite(); err != nil {
		return 0, "", "", fmt.Errorf("закрытие stdin: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, hijacked.Reader); err != nil {
		return 0, "", "", fmt.Errorf("чтение вывода: %w", err)
	}

	inspect, err := d.cli.ContainerExecInspect(execCtx, execResp.ID)
	if err != nil {
		return 0, "", "", fmt.Errorf("exec inspect: %w", err)
	}

	return inspect.ExitCode, stdout.String(), stderr.String(), nil
}

// normalizeOutput приводит вывод к каноническому виду перед сравнением:
// убирает пробелы/табы в конце строк и пустые строки в конце вывода, чтобы
// не засчитывать WA из-за незначащих различий в форматировании.
func normalizeOutput(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func errorResult(submissionID, msg string) *judgepb.SubmitResultRequest {
	return &judgepb.SubmitResultRequest{
		SubmissionId: submissionID,
		Status:       judgepb.Status_STATUS_ERROR,
		Error:        msg,
	}
}
