package meetingRecord

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"doubao-speech-service/internal/service/media"
)

type recordOptions struct {
	Dir             string
	MaxBytes        int64
	UploadQueueSize int
	SampleRate      int
	Channels        int
	BitsPerSample   int // 注意默认值 16 对应 s16le，如果要改，startConvertWorkers 里面对 Convert 方法传递的参数也更改
	ConvertEnabled  bool
	ConvertFormat   string
	ConvertBitrate  string
	FFmpegPath      string
	ConvertWorkers  int
}

type convertResult struct {
	convertedPath string
	err           error
}
type convertTask struct {
	recorder *Recorder
	resultCh chan convertResult
}

var (
	options         recordOptions
	uploadQueue     chan RecordingResult
	convertQueue    chan convertTask
	formatConverter *media.FFmpegConverter
)

func init() {
	ctx := context.Background()

	tryFFMpeg, _ := exec.LookPath("ffmpeg")
	options = recordOptions{
		Dir:             g.Cfg().MustGet(ctx, "meeting.record.dir", "/app/uploads").String(),
		MaxBytes:        g.Cfg().MustGet(ctx, "meeting.record.maxBytes", 268435456).Int64(),
		SampleRate:      g.Cfg().MustGet(ctx, "meeting.record.sampleRate", 16000).Int(),
		Channels:        g.Cfg().MustGet(ctx, "meeting.record.channels", 1).Int(),
		BitsPerSample:   g.Cfg().MustGet(ctx, "meeting.record.bitsPerSample", 16).Int(),
		ConvertEnabled:  g.Cfg().MustGet(ctx, "meeting.record.convert.enabled", true).Bool(),
		ConvertFormat:   g.Cfg().MustGet(ctx, "meeting.record.convert.format", "ogg").String(),
		ConvertBitrate:  g.Cfg().MustGet(ctx, "meeting.record.convert.bitrate", "64k").String(),
		FFmpegPath:      g.Cfg().MustGet(ctx, "meeting.record.convert.ffmpeg", tryFFMpeg).String(),
		UploadQueueSize: g.Cfg().MustGet(ctx, "meeting.record.upload.queueSize", 1).Int(),
		ConvertWorkers:  g.Cfg().MustGet(ctx, "meeting.record.convert.workers", 5).Int(),
	}

	// 初始化格式转换器
	if options.ConvertEnabled {
		formatConverter = media.NewConverter(options.FFmpegPath, media.ConvertOptions{
			TargetFormat: options.ConvertFormat,
			AudioBitrate: options.ConvertBitrate,
			DeleteInput:  true,
		})
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

// 获取转换器。这个值可能是 nil，例如当 options.ConvertEnabled = false 时。
func getConverter() *media.FFmpegConverter {
	return formatConverter
}

// submitConvertTask 提交转换任务到队列，返回结果 channel。
// ConvertWorkers 会智能处理没有开启转换器的情况，外层函数调用时无需特殊判断。
func submitConvertTask(r *Recorder) convertResult {
	resultCh := make(chan convertResult, 1)
	task := convertTask{
		recorder: r,
		resultCh: resultCh,
	}
	// 发送任务到队列
	convertQueue <- task
	// 等待结果返回
	return <-resultCh
}

// startConvertWorkers 启动转换 worker goroutines
func startConvertWorkers(ctx context.Context, numWorkers int) {
	for range numWorkers {
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-convertQueue:
					g.Log().Infof(task.recorder.ctx, "收到转换任务，文件: %s", task.recorder.filePath)
					conv := getConverter()
					if conv == nil {
						// 说明关掉了 Converter，此时应该执行 wav 保存
						g.Log().Infof(task.recorder.ctx, "转换器未启用，执行 PCM -> WAV 转换")
						if err := task.recorder.convertToWAV(); err != nil {
							g.Log().Errorf(task.recorder.ctx, "转换成WAV失败: %v", err)
							task.resultCh <- convertResult{convertedPath: "", err: gerror.Wrap(err, "转换成WAV失败")}
							continue
						}
						g.Log().Infof(task.recorder.ctx, "WAV 转换完成")
						task.resultCh <- convertResult{convertedPath: task.recorder.filePath, err: nil}
						continue
					} else {
						// 正常执行转换任务
						g.Log().Infof(task.recorder.ctx, "开始 FFmpeg 转换，参数: f=s16le, ar=%d, ac=%d", task.recorder.sampleRate, task.recorder.channels)
						convertedPath, err := conv.Convert(task.recorder.ctx, task.recorder.filePath, g.MapStrStr{
							"f":  "s16le",
							"ar": strconv.Itoa(task.recorder.sampleRate),
							"ac": strconv.Itoa(task.recorder.channels),
						})
						if err != nil {
							g.Log().Errorf(task.recorder.ctx, "FFmpeg 转换失败: %v", err)
							task.resultCh <- convertResult{convertedPath: "", err: err}
							continue
						}
						g.Log().Infof(task.recorder.ctx, "FFmpeg 转换完成: %s", convertedPath)
						task.recorder.filePath = convertedPath
						task.resultCh <- convertResult{convertedPath: convertedPath, err: nil}
					}
				}
			}
		}(ctx)
	}
	g.Log().Infof(ctx, "成功启动 %d 个转换 workers", numWorkers)
}
