package meetingRecord

import (
	"context"
	"os"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
)

type recordOptions struct {
	Dir             string
	MaxBytes        int64
	UploadQueueSize int
	SampleRate      int
	Channels        int
	BitsPerSample   int
}

var (
	optionsMu sync.RWMutex
	options   recordOptions

	uploadQueue chan RecordingResult
)

// Init 读取配置，并在需要时启动上传 worker。
func Init(ctx context.Context) error {
	cfg := g.Cfg()
	opts := recordOptions{
		Dir:             cfg.MustGet(ctx, "meeting.record.dir").String(),
		MaxBytes:        cfg.MustGet(ctx, "meeting.record.maxBytes").Int64(),
		UploadQueueSize: cfg.MustGet(ctx, "meeting.record.upload.queueSize").Int(),
		SampleRate:      cfg.MustGet(ctx, "meeting.record.sampleRate").Int(),
		Channels:        cfg.MustGet(ctx, "meeting.record.channels").Int(),
		BitsPerSample:   cfg.MustGet(ctx, "meeting.record.bitsPerSample").Int(),
	}

	if opts.SampleRate == 0 {
		opts.SampleRate = 16000
	}
	if opts.Channels == 0 {
		opts.Channels = 1
	}
	if opts.BitsPerSample == 0 {
		opts.BitsPerSample = 16
	}

	if opts.Dir != "" {
		if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
			return err
		}
	}
	if opts.UploadQueueSize <= 0 {
		opts.UploadQueueSize = 1
	}

	optionsMu.Lock()
	options = opts
	optionsMu.Unlock()

	if uploadQueue == nil {
		uploadQueue = make(chan RecordingResult, opts.UploadQueueSize*2+2)
		startUploadWorkers(ctx, opts)
	}
	return nil
}

func getOptions() recordOptions {
	optionsMu.RLock()
	defer optionsMu.RUnlock()
	return options
}
