package worker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const (
	testTimeLimit    = 5 * time.Second
	compileTimeLimit = 40 * time.Second
	memoryLimitMiB   = 256
	execPidsLimit    = 64
	execCPULimit     = 1_000_000_000   // 1 vCPU, в единицах NanoCPUs
	dockerCallSlack  = 2 * time.Second // запас поверх лимита на накладные расходы Docker API
	imagePullTimeout = 30 * time.Minute
)

type languageSpec struct {
	image          string
	sourceFile     string
	compileCmd     string
	runCmd         string
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

type DockerJudge struct {
	cli *client.Client

	mu    sync.Mutex
	pulls map[string]*imagePull
}

type imagePull struct {
	done chan struct{}
	err  error
}

func NewDockerJudge() (*DockerJudge, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &DockerJudge{cli: cli, pulls: make(map[string]*imagePull)}, nil
}

func (d *DockerJudge) WarmImages(ctx context.Context) {
	seen := make(map[string]bool, len(languageSpecs))
	for _, spec := range languageSpecs {
		if seen[spec.image] {
			continue
		}
		seen[spec.image] = true

		go func(img string) {
			start := time.Now()
			log.Printf("[judge] подгружаю образ %s в фоне", img)
			if err := d.ensureImage(ctx, img); err != nil {
				log.Printf("[judge] образ %s не готов (%v), повторю при первом сабмишне на этом языке", img, err)
				return
			}
			log.Printf("[judge] образ %s готов (%s)", img, time.Since(start).Round(time.Second))
		}(spec.image)
	}
}

func (d *DockerJudge) ensureImage(ctx context.Context, img string) error {
	d.mu.Lock()
	p, inFlight := d.pulls[img]
	if !inFlight {
		p = &imagePull{done: make(chan struct{})}
		d.pulls[img] = p
		go d.pull(img, p)
	}
	d.mu.Unlock()

	if inFlight {
		select {
		case <-p.done:
			return p.err
		default:
			log.Printf("[judge] образ %s ещё подгружается, жду", img)
		}
	}

	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *DockerJudge) pull(img string, p *imagePull) {

	ctx, cancel := context.WithTimeout(context.Background(), imagePullTimeout)
	defer cancel()

	p.err = d.pullImage(ctx, img)
	close(p.done)

	if p.err != nil {
		d.mu.Lock()
		delete(d.pulls, img)
		d.mu.Unlock()
	}
}

func (d *DockerJudge) pullImage(ctx context.Context, img string) error {
	if _, err := d.cli.ImageInspect(ctx, img); err == nil {
		return nil
	} else if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect image %s: %w", img, err)
	}

	rc, err := d.cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", img, err)
	}
	defer rc.Close() //nolint:errcheck

	if _, err := io.Copy(io.Discard, rc); err != nil {
		return fmt.Errorf("pull image %s: %w", img, err)
	}
	return nil
}

func (d *DockerJudge) Judge(ctx context.Context, task *judgepb.Task) *judgepb.SubmitResultRequest {
	spec, ok := languageSpecs[task.Language]
	if !ok {
		return errorResult(task.SubmissionId, fmt.Sprintf("неподдерживаемый язык: %q", task.Language))
	}

	if err := d.ensureImage(ctx, spec.image); err != nil {
		return errorResult(task.SubmissionId, fmt.Sprintf("образ %s недоступен: %v", spec.image, err))
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
		return judgepb.Status_STATUS_TL, "", "", nil
	case 137:
		return judgepb.Status_STATUS_ML, "", "", nil
	default:
		return judgepb.Status_STATUS_RE, stdout, stderr, nil
	}
}

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
