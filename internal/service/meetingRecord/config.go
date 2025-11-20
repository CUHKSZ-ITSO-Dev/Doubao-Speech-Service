package meetingRecord

import (
	"context"
	"os"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
)

type recordOptions struct {
	Dir               string
	MaxBytes          int64
	UploadEndpoint    string
	UploadConcurrency int
	SampleRate        int
	Channels          int
	BitsPerSample     int
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
		Dir:               cfg.MustGet(ctx, "meeting.record.dir").String(),
		MaxBytes:          cfg.MustGet(ctx, "meeting.record.maxBytes").Int64(),
		UploadEndpoint:    cfg.MustGet(ctx, "meeting.record.upload.endpoint").String(),
		UploadConcurrency: cfg.MustGet(ctx, "meeting.record.upload.concurrency").Int(),
		SampleRate:        cfg.MustGet(ctx, "meeting.record.sampleRate").Int(),
		Channels:          cfg.MustGet(ctx, "meeting.record.channels").Int(),
		BitsPerSample:     cfg.MustGet(ctx, "meeting.record.bitsPerSample").Int(),
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
	if opts.UploadConcurrency <= 0 {
		opts.UploadConcurrency = 1
	}

	optionsMu.Lock()
	options = opts
	optionsMu.Unlock()

	if opts.UploadEndpoint != "" && uploadQueue == nil {
		uploadQueue = make(chan RecordingResult, opts.UploadConcurrency*2+2)
		startUploadWorkers(ctx, opts)
	}
	return nil
}

func getOptions() recordOptions {
	optionsMu.RLock()
	defer optionsMu.RUnlock()
	return options
}
