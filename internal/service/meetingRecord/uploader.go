package meetingRecord

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gogf/gf/v2/frame/g"

	"doubao-speech-service/internal/service/volcengine"
)

var uploadOnce sync.Once

func startUploadWorkers(ctx context.Context, opts recordOptions) {
	uploadOnce.Do(func() {
		for i := 0; i < opts.UploadQueueSize; i++ {
			go uploadWorker(ctx)
		}
	})
}

func uploadWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-uploadQueue:
			if err := uploadOne(ctx, item); err != nil {
				g.Log().Warningf(ctx, "record upload failed, connect_id=%s: %v", item.ConnectID, err)
				continue
			}
			g.Log().Infof(ctx, "record upload completed, connect_id=%s, size=%d bytes", item.ConnectID, item.Size)
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

	uploadFile, err := volcengine.NewLocalUploadFile(item.FilePath)
	if err != nil {
		return err
	}
	res := volcengine.ProcessFileUpload(ctx, uploadFile, item.Owner)
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
