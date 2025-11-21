package meetingRecord

import (
	"context"
	"os"
	"sync"

	"github.com/gogf/gf/v2/frame/g"

	"doubao-speech-service/internal/service/media"
)

type recordOptions struct {
	Dir             string
	MaxBytes        int64
	UploadQueueSize int
	SampleRate      int
	Channels        int
	BitsPerSample   int
	ConvertEnabled  bool
	ConvertFormat   string
	ConvertBitrate  string
	FFmpegPath      string
}

var (
	optionsMu sync.RWMutex
	options   recordOptions

	uploadQueue     chan RecordingResult
	converterMu     sync.RWMutex
	formatConverter media.FormatConverter
)

// Init 读取配置，并在需要时启动上传 worker。
func Init(ctx context.Context) error {
	cfg := g.Cfg()
	convertEnabledVar, _ := cfg.Get(ctx, "meeting.record.convert.enabled")
	convertEnabled := true
	if convertEnabledVar != nil && !convertEnabledVar.IsNil() {
		convertEnabled = convertEnabledVar.Bool()
	}
	opts := recordOptions{
		Dir:             cfg.MustGet(ctx, "meeting.record.dir").String(),
		MaxBytes:        cfg.MustGet(ctx, "meeting.record.maxBytes").Int64(),
		UploadQueueSize: cfg.MustGet(ctx, "meeting.record.upload.queueSize").Int(),
		SampleRate:      cfg.MustGet(ctx, "meeting.record.sampleRate").Int(),
		Channels:        cfg.MustGet(ctx, "meeting.record.channels").Int(),
		BitsPerSample:   cfg.MustGet(ctx, "meeting.record.bitsPerSample").Int(),
		ConvertEnabled:  convertEnabled,
		ConvertFormat:   cfg.MustGet(ctx, "meeting.record.convert.format").String(),
		ConvertBitrate:  cfg.MustGet(ctx, "meeting.record.convert.bitrate").String(),
		FFmpegPath:      cfg.MustGet(ctx, "meeting.record.convert.ffmpeg").String(),
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
	if opts.ConvertFormat == "" {
		opts.ConvertFormat = "ogg"
	}
	if opts.ConvertEnabled && opts.ConvertBitrate == "" {
		opts.ConvertBitrate = "64k"
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

	if opts.ConvertEnabled {
		conv, err := media.NewFFmpegConverter(opts.FFmpegPath, media.ConvertOptions{
			TargetFormat: opts.ConvertFormat,
			AudioBitrate: opts.ConvertBitrate,
			DeleteInput:  true,
		})
		if err != nil {
			g.Log().Warningf(ctx, "meeting record converter init failed, wav will be kept: %v", err)
		}
		setConverter(conv)
	} else {
		setConverter(nil)
	}

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

func getConverter() media.FormatConverter {
	converterMu.RLock()
	defer converterMu.RUnlock()
	return formatConverter
}

func setConverter(c media.FormatConverter) {
	converterMu.Lock()
	defer converterMu.Unlock()
	formatConverter = c
}
