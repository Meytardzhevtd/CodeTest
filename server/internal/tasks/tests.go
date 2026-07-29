package tasks

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

const (
	// MaxTestsArchiveBytes caps the total size of an uploaded tests zip.
	MaxTestsArchiveBytes = 50 << 20 // 50 MiB
	// maxTestFileBytes caps the uncompressed size of a single .in/.out file,
	// checked before reading so a crafted zip can't decompression-bomb us.
	maxTestFileBytes = 10 << 20 // 10 MiB
	// maxTestsCount caps how many test pairs one task can have.
	maxTestsCount = 500
)

// testFileNamePattern matches the file names task authors are expected to
// use inside the tests archive, e.g. "test_001.in" / "test_001.out". This is
// intentionally independent of the key format pkg/storage uses internally
// (see storage.testKey) - the two naming schemes don't need to match.
var testFileNamePattern = regexp.MustCompile(`^test_(\d+)\.(in|out)$`)

type zipTestPair struct {
	in, out *zip.File
}

// UploadTests replaces the entire test set for a task from a zip archive of
// test_NNN.in / test_NNN.out files. Only the task's creator may do this.
// It talks only to object storage - no rows are read or written except the
// read-only ownership check against the task itself.
func (s *Service) UploadTests(ctx context.Context, userID, taskID string, archive io.ReaderAt, size int64) (int, error) {
	if taskID == "" {
		return 0, fmt.Errorf("%w: task id cannot be empty", ErrInvalidArchive)
	}
	if size <= 0 || size > MaxTestsArchiveBytes {
		return 0, fmt.Errorf("%w: archive size must be between 1 and %d bytes", ErrInvalidArchive, MaxTestsArchiveBytes)
	}

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		if err == ErrTaskNotFound {
			return 0, ErrTaskNotFound
		}
		return 0, fmt.Errorf("get task by id: %w", err)
	}
	if task.CreatedBy != userID {
		return 0, ErrForbidden
	}

	zr, err := zip.NewReader(archive, size)
	if err != nil {
		return 0, fmt.Errorf("%w: not a valid zip file: %v", ErrInvalidArchive, err)
	}

	pairs, err := parseTestsArchive(zr)
	if err != nil {
		return 0, err
	}

	numbers := make([]int, 0, len(pairs))
	for num := range pairs {
		numbers = append(numbers, num)
	}
	sort.Ints(numbers)

	// Replace-all semantics: wipe whatever tests exist today before writing
	// the new set, so we never end up with a mix of old and new tests.
	if err := s.store.DeleteTask(ctx, taskID); err != nil {
		return 0, fmt.Errorf("clear existing tests: %w", err)
	}

	for _, num := range numbers {
		if err := uploadTestPair(ctx, s.store, taskID, num, pairs[num]); err != nil {
			// Best-effort rollback: don't leave a partially-replaced test set.
			_ = s.store.DeleteTask(ctx, taskID)
			return 0, fmt.Errorf("upload test %03d: %w", num, err)
		}
	}

	return len(numbers), nil
}

func parseTestsArchive(zr *zip.Reader) (map[int]zipTestPair, error) {
	pairs := make(map[int]zipTestPair)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if strings.Contains(f.Name, "..") {
			return nil, fmt.Errorf("%w: unsafe entry name %q", ErrInvalidArchive, f.Name)
		}

		base := path.Base(f.Name) // zip entries always use "/", regardless of OS
		if isIgnoredArchiveEntry(f.Name, base) {
			continue
		}

		m := testFileNamePattern.FindStringSubmatch(base)
		if m == nil {
			return nil, fmt.Errorf("%w: unrecognized file %q, expected test_NNN.in or test_NNN.out", ErrInvalidArchive, f.Name)
		}

		num, err := strconv.Atoi(m[1])
		if err != nil || num < 1 {
			return nil, fmt.Errorf("%w: invalid test number in %q", ErrInvalidArchive, f.Name)
		}
		if f.UncompressedSize64 > maxTestFileBytes {
			return nil, fmt.Errorf("%w: %q exceeds max size of %d bytes", ErrInvalidArchive, f.Name, maxTestFileBytes)
		}

		p := pairs[num]
		if m[2] == "in" {
			p.in = f
		} else {
			p.out = f
		}
		pairs[num] = p

		if len(pairs) > maxTestsCount {
			return nil, fmt.Errorf("%w: archive has more than %d tests", ErrInvalidArchive, maxTestsCount)
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("%w: archive contains no test files", ErrInvalidArchive)
	}

	for num, p := range pairs {
		if p.in == nil || p.out == nil {
			return nil, fmt.Errorf("%w: test %03d is incomplete (in=%v out=%v)", ErrInvalidArchive, num, p.in != nil, p.out != nil)
		}
	}

	return pairs, nil
}

func isIgnoredArchiveEntry(name, base string) bool {
	return strings.HasPrefix(base, "._") ||
		base == ".DS_Store" ||
		strings.HasPrefix(name, "__MACOSX/")
}

func uploadTestPair(ctx context.Context, store storage.Writer, taskID string, num int, p zipTestPair) error {
	inRC, err := p.in.Open()
	if err != nil {
		return fmt.Errorf("open %q: %w", p.in.Name, err)
	}
	defer inRC.Close()

	outRC, err := p.out.Open()
	if err != nil {
		return fmt.Errorf("open %q: %w", p.out.Name, err)
	}
	defer outRC.Close()

	return store.UploadTest(ctx, taskID, num,
		inRC, int64(p.in.UncompressedSize64),
		outRC, int64(p.out.UncompressedSize64),
	)
}
