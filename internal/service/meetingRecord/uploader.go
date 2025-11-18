package meetingRecord

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

var uploadOnce sync.Once

func startUploadWorkers(ctx context.Context, opts recordOptions) {
	uploadOnce.Do(func() {
		for i := 0; i < opts.UploadConcurrency; i++ {
			go uploadWorker(ctx, opts)
		}
	})
}

func uploadWorker(ctx context.Context, opts recordOptions) {
	logger := g.Log()
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-uploadQueue:
			if err := uploadOne(ctx, client, item, opts); err != nil {
				logger.Warningf(ctx, "record upload failed, connect_id=%s: %v", item.ConnectID, err)
				continue
			}
			logger.Infof(ctx, "record upload completed, connect_id=%s, size=%d bytes", item.ConnectID, item.Size)
			_ = os.Remove(item.FilePath)
			_ = os.Remove(item.Dir)
		}
	}
}

func uploadOne(ctx context.Context, client *http.Client, item RecordingResult, opts recordOptions) error {
	fileInfo, err := os.Stat(item.FilePath)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() || fileInfo.Size() == 0 {
		return fmt.Errorf("invalid recording file: %s", item.FilePath)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	contentType := writer.FormDataContentType()

	go func() {
		defer pw.Close()
		defer writer.Close()

		file, err := os.Open(item.FilePath)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer file.Close()

		part, err := writer.CreateFormFile("files", filepath.Base(item.FilePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err = io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.UploadEndpoint, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("upload failed with status %s: %s", resp.Status, string(respBody))
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

// EnqueueUpload 将录音结果加入上传队列。
func EnqueueUpload(ctx context.Context, result *RecordingResult) {
	if result == nil || uploadQueue == nil {
		return
	}
	select {
	case uploadQueue <- *result:
	case <-ctx.Done():
	default:
		uploadQueue <- *result
	}
}
