package storage

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"

	"github.com/minio/minio-go/v7"
)

// \d+ (not a fixed width) so test numbers past the zero-padding width used by
// testKey are still recognized instead of silently dropped from listings.
var keyPattern = regexp.MustCompile(`^test(\d+)\.(in|out)$`)

type TestCase struct {
	Number int
	InKey  string
	OutKey string
}

func testKey(taskID string, testNum int, ext string) string {
	return fmt.Sprintf("%s/test%03d.%s", taskID, testNum, ext)
}

func (c *Client) UploadTest(ctx context.Context, taskID string, testNum int, input io.Reader, inputSize int64, output io.Reader, outputSize int64) error {
	inKey := testKey(taskID, testNum, "in")
	outKey := testKey(taskID, testNum, "out")

	if _, err := c.mc.PutObject(ctx, c.bucket, inKey, input, inputSize, minio.PutObjectOptions{
		ContentType: "text/plain",
	}); err != nil {
		return fmt.Errorf("storage: upload %q: %w", inKey, err)
	}

	if _, err := c.mc.PutObject(ctx, c.bucket, outKey, output, outputSize, minio.PutObjectOptions{
		ContentType: "text/plain",
	}); err != nil {
		// Best-effort cleanup: don't leave an orphaned .in with no matching
		// .out, which would otherwise fail every future ListTests for this task.
		if rmErr := c.mc.RemoveObject(ctx, c.bucket, inKey, minio.RemoveObjectOptions{}); rmErr != nil {
			return fmt.Errorf("storage: upload %q: %w (cleanup of %q also failed: %v)", outKey, err, inKey, rmErr)
		}
		return fmt.Errorf("storage: upload %q: %w", outKey, err)
	}

	return nil
}

func (c *Client) ListTests(ctx context.Context, taskID string) ([]TestCase, error) {
	prefix := taskID + "/"

	type pair struct {
		hasIn, hasOut bool
		inKey, outKey string
	}
	byNumber := make(map[int]*pair)

	objCh := c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("storage: list objects for task %q: %w", taskID, obj.Err)
		}

		name := obj.Key[len(prefix):]
		m := keyPattern.FindStringSubmatch(name)
		if m == nil {
			continue
		}

		num, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("storage: parse test number from key %q: %w", obj.Key, err)
		}

		p, ok := byNumber[num]
		if !ok {
			p = &pair{}
			byNumber[num] = p
		}
		if m[2] == "in" {
			p.hasIn = true
			p.inKey = obj.Key
		} else {
			p.hasOut = true
			p.outKey = obj.Key
		}
	}

	if len(byNumber) == 0 {
		return nil, fmt.Errorf("storage: no tests found for task %q", taskID)
	}

	numbers := make([]int, 0, len(byNumber))
	for num := range byNumber {
		numbers = append(numbers, num)
	}
	sort.Ints(numbers)

	result := make([]TestCase, 0, len(numbers))
	for _, num := range numbers {
		p := byNumber[num]
		if !p.hasIn || !p.hasOut {
			return nil, fmt.Errorf("storage: task %q test %03d is incomplete (in=%v out=%v)", taskID, num, p.hasIn, p.hasOut)
		}
		result = append(result, TestCase{
			Number: num,
			InKey:  p.inKey,
			OutKey: p.outKey,
		})
	}

	return result, nil
}

func (c *Client) GetTest(ctx context.Context, tc TestCase) (input io.ReadCloser, output io.ReadCloser, err error) {
	inObj, err := c.mc.GetObject(ctx, c.bucket, tc.InKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("storage: get %q: %w", tc.InKey, err)
	}
	// GetObject is lazy: it doesn't hit the network until the first read/stat,
	// so a missing key only surfaces here, not on the call above.
	if _, err := inObj.Stat(); err != nil {
		inObj.Close()
		return nil, nil, fmt.Errorf("storage: stat %q: %w", tc.InKey, err)
	}

	outObj, err := c.mc.GetObject(ctx, c.bucket, tc.OutKey, minio.GetObjectOptions{})
	if err != nil {
		inObj.Close()
		return nil, nil, fmt.Errorf("storage: get %q: %w", tc.OutKey, err)
	}
	if _, err := outObj.Stat(); err != nil {
		inObj.Close()
		outObj.Close()
		return nil, nil, fmt.Errorf("storage: stat %q: %w", tc.OutKey, err)
	}

	return inObj, outObj, nil
}

func (c *Client) DeleteTask(ctx context.Context, taskID string) error {
	prefix := taskID + "/"

	objCh := c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	errCh := c.mc.RemoveObjects(ctx, c.bucket, objCh, minio.RemoveObjectsOptions{})
	for rmErr := range errCh {
		if rmErr.Err != nil {
			return fmt.Errorf("storage: delete object %q: %w", rmErr.ObjectName, rmErr.Err)
		}
	}
	return nil
}
