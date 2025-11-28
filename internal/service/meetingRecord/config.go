package meetingRecord

import (
	"context"
	"os"

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
	ConvertWorkers  int
}

type convertTask struct {
	ctx      context.Context
	wavPath  string
	resultCh chan<- convertResult
}

type convertResult struct {
	path string
	err  error
}

var (
	options         recordOptions
	uploadQueue     chan RecordingResult
	convertQueue    chan convertTask
	formatConverter *media.FFmpegConverter
)

func init() {
	ctx := context.Background()

	convertEnabled := g.Cfg().MustGet(ctx, "meeting.record.convert.enabled", false).Bool()
	options = recordOptions{
		Dir:             g.Cfg().MustGet(ctx, "meeting.record.dir").String(),
		MaxBytes:        g.Cfg().MustGet(ctx, "meeting.record.maxBytes").Int64(),
		UploadQueueSize: g.Cfg().MustGet(ctx, "meeting.record.upload.queueSize").Int(),
		SampleRate:      g.Cfg().MustGet(ctx, "meeting.record.sampleRate").Int(),
		Channels:        g.Cfg().MustGet(ctx, "meeting.record.channels").Int(),
		BitsPerSample:   g.Cfg().MustGet(ctx, "meeting.record.bitsPerSample").Int(),
		ConvertEnabled:  convertEnabled,
		ConvertFormat:   g.Cfg().MustGet(ctx, "meeting.record.convert.format").String(),
		ConvertBitrate:  g.Cfg().MustGet(ctx, "meeting.record.convert.bitrate").String(),
		FFmpegPath:      g.Cfg().MustGet(ctx, "meeting.record.convert.ffmpeg").String(),
		ConvertWorkers:  g.Cfg().MustGet(ctx, "meeting.record.convert.workers", 5).Int(),
	}

	// 设置默认值
	if options.SampleRate == 0 {
		options.SampleRate = 16000
	}
	if options.Channels == 0 {
		options.Channels = 1
	}
	if options.BitsPerSample == 0 {
		options.BitsPerSample = 16
	}
	if options.ConvertFormat == "" {
		options.ConvertFormat = "ogg"
	}
	if options.ConvertEnabled && options.ConvertBitrate == "" {
		options.ConvertBitrate = "64k"
	}
	if options.UploadQueueSize <= 0 {
		options.UploadQueueSize = 1
	}
	if options.ConvertWorkers <= 0 {
		options.ConvertWorkers = 5
	}

	// 创建录音目录
	if options.Dir != "" {
		if err := os.MkdirAll(options.Dir, 0o755); err != nil {
			g.Log().Warningf(ctx, "meeting record dir creation failed, recording disabled: %v", err)
			options.Dir = "" // 禁用录音功能
		}
	}

	// 初始化格式转换器
	if options.ConvertEnabled {
		conv, err := media.NewFFmpegConverter(options.FFmpegPath, media.ConvertOptions{
			TargetFormat: options.ConvertFormat,
			AudioBitrate: options.ConvertBitrate,
			DeleteInput:  true,
		})
		if err != nil {
			g.Log().Warningf(ctx, "meeting record converter init failed, wav will be kept: %v", err)
		}
		formatConverter = conv

		// 初始化转换队列和启动 workers
		convertQueue = make(chan convertTask, options.ConvertWorkers*2)
		startConvertWorkers(ctx, options.ConvertWorkers)
	}

	// 初始化上传队列和启动 workers
	uploadQueue = make(chan RecordingResult, options.UploadQueueSize*2+2)
	startUploadWorkers(ctx, options)
}

func getOptions() recordOptions {
	return options
}

func getConverter() *media.FFmpegConverter {
	return formatConverter
}

// submitConvertTask 提交转换任务到队列，返回结果 channel
func submitConvertTask(ctx context.Context, wavPath string) <-chan convertResult {
	resultCh := make(chan convertResult, 1)
	if convertQueue == nil {
		//如果 convertQueue 未初始化（即为 nil），说明没有任务队列，直接“假装”任务完成。
		resultCh <- convertResult{path: wavPath, err: nil}
		return resultCh
	}

	task := convertTask{
		ctx:      ctx,
		wavPath:  wavPath,
		resultCh: resultCh,
	}

	select {
	case convertQueue <- task:
	case <-ctx.Done():
		resultCh <- convertResult{path: wavPath, err: ctx.Err()}
	}

	return resultCh
}

// startConvertWorkers 启动转换 worker goroutines
func startConvertWorkers(ctx context.Context, numWorkers int) {
	for range numWorkers {
		go convertWorker(ctx)
	}
	g.Log().Infof(ctx, "started %d convert workers", numWorkers)
}

// convertWorker 转换任务处理器
func convertWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-convertQueue:
			conv := getConverter()
			if conv == nil {
				task.resultCh <- convertResult{path: task.wavPath, err: nil}
				continue
			}

			convertedPath, err := conv.Convert(task.ctx, task.wavPath)
			task.resultCh <- convertResult{path: convertedPath, err: err}
		}
	}
}
