package meetingRecord

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"

	"github.com/gogf/gf/v2/frame/g"

	transcriptionCtl "doubao-speech-service/internal/controller/transcription"
)

var uploadOnce sync.Once

func startUploadWorkers(ctx context.Context, opts recordOptions) {
	uploadOnce.Do(func() {
		for i := 0; i < opts.UploadConcurrency; i++ {
			go uploadWorker(ctx)
		}
	})
}

func uploadWorker(ctx context.Context) {
	logger := g.Log()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-uploadQueue:
			if err := uploadOne(ctx, item); err != nil {
				logger.Warningf(ctx, "record upload failed, connect_id=%s: %v", item.ConnectID, err)
				continue
			}
			logger.Infof(ctx, "record upload completed, connect_id=%s, size=%d bytes", item.ConnectID, item.Size)
			_ = os.Remove(item.FilePath)
			_ = os.Remove(item.Dir)
		}
	}
}

func uploadOne(ctx context.Context, item RecordingResult) error {
	fileInfo, err := os.Stat(item.FilePath)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() || fileInfo.Size() == 0 {
		return fmt.Errorf("invalid recording file: %s", item.FilePath)
	}

	if item.Owner == "" {
		return fmt.Errorf("missing owner for recording: %s", item.ConnectID)
	}

	res := transcriptionCtl.ProcessFileUpload(ctx, &recordingUploadFile{
		path: item.FilePath,
		size: fileInfo.Size(),
	}, item.Owner)
	return res.Error
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

type recordingUploadFile struct {
	path string
	size int64
}

func (r *recordingUploadFile) FileName() string {
	return filepath.Base(r.path)
}

func (r *recordingUploadFile) FileSize() int64 {
	return r.size
}

func (r *recordingUploadFile) Open() (multipart.File, error) {
	return os.Open(r.path)
}
