package meetingRecord

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"doubao-speech-service/internal/service/media"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var (
	ErrRecorderDisabled = errors.New("recorder disabled")
)

// IsRecorderDisabled 检查错误是否为录音器被禁用错误
func IsRecorderDisabled(err error) bool {
	return errors.Is(err, ErrRecorderDisabled)
}

// Recorder 按连接记录客户端发送的音频帧。
type Recorder struct {
	ctx           context.Context        // 每个请求都会开一个 Recorder，ctx 就是请求的 context
	mu            sync.Mutex             // 互斥锁，保护文件操作
	fileIO        *os.File               // 文件 IO 流，已打开，应该写入 PCM 音频流。当 fileIO == nil 时，说明文件已经被关闭且不该继续写入。
	dir           string                 // /opts.Dir/2006_01_02/ctxId/
	filePath      string                 // /opts.Dir/2006_01_02/ctxId/Meeting_2006_01_02_150405.pcm
	total         int64                  // 当前文件的大小。会随着 Append 不断增加。
	maxBytes      int64                  // 最大字节数
	startTime     time.Time              // 开始时间
	sampleRate    int                    // 采样率
	bitsPerSample int                    // 位深度
	channels      int                    // 通道数
	converter     *media.FFmpegConverter // 转换器
}

// RecordingResult 提供录音文件的元数据，供上传流程使用。
type RecordingResult struct {
	ConnectID string
	Owner     string
	FilePath  string
	Dir       string
	Size      int64
	StartedAt time.Time
	EndedAt   time.Time
}

func NewRecorder(ctx context.Context) (*Recorder, error) {
	ctxId := gctx.CtxId(ctx)
	opts := getOptions()
	if opts.Dir == "" {
		return nil, ErrRecorderDisabled
	}
	now := time.Now()
	formattedDate := now.Format("2006_01_02")
	formattedTime := now.Format("2006_01_02_150405")

	// 创建目录
	dir := filepath.Join(opts.Dir, formattedDate, ctxId)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, gerror.Wrap(err, "创建目录失败")
	}
	// 创建 PCM 文件
	path := filepath.Join(dir, "Meeting_"+formattedTime+".pcm")
	if _, err := os.Create(path); err != nil {
		return nil, gerror.Wrap(err, "创建文件失败")
	}

	return &Recorder{
		ctx:           ctx,
		dir:           dir,
		filePath:      path,
		startTime:     now,
		maxBytes:      opts.MaxBytes,
		sampleRate:    opts.SampleRate,
		bitsPerSample: opts.BitsPerSample,
		channels:      opts.Channels,
		converter:     getConverter(),
	}, nil
}

// Append 写入一帧音频。若超过限制或写入失败，会终止录制。
func (r *Recorder) Append(frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fileIO == nil {
		return gerror.New("文件流已经关闭，不能够继续写入音频帧。")
	}
	if r.total+int64(len(frame)) > r.maxBytes {
		if err := r.fileIO.Close(); err != nil {
			g.Log().Errorf(r.ctx, "超过最大字节数限制，拒绝继续写入，强制关闭文件。但是关闭文件流时发生失败。%v", err)
			return gerror.Wrap(err, "超过最大字节数限制，拒绝继续写入，强制关闭文件。但是关闭文件流时发生失败。")
		}
		r.fileIO = nil
		g.Log().Errorf(r.ctx, "超过最大字节数限制，拒绝继续写入，强制关闭文件。")
		return gerror.New("超过最大字节数限制，拒绝继续写入，强制关闭文件。")
	}
	n, err := r.fileIO.Write(frame)
	if err != nil {
		g.Log().Errorf(r.ctx, "追加音频流时发生错误。%v", err)
		return gerror.Wrap(err, "追加音频流时发生错误。")
	}
	r.total += int64(n)
	return nil
}

// Finalize 结束录制。返回 nil 表示没有有效数据。
func (r *Recorder) Finalize() (*RecordingResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fileIO != nil {
		if err := r.fileIO.Close(); err != nil {
			return nil, gerror.Wrap(err, "关闭文件失败")
		}
		r.fileIO = nil
	}
	if r.total == 0 {
		if err := os.Remove(r.filePath); err != nil {
			g.Log().Errorf(r.ctx, "关闭文件流后，因为总大小为 0，删除文件。但是删除文件失败。%v", err)
			return nil, gerror.Wrap(err, "关闭文件流后，因为总大小为 0，删除文件。但是删除文件失败。")
		}
		g.Log().Info(r.ctx, "关闭文件流后，因为总大小为 0，删除文件。")
		return nil, nil
	}

	// TODO: convertResult 的文件路径好像没有什么用
	if convertResult := submitConvertTask(r); convertResult.err != nil {
		return nil, gerror.Wrap(convertResult.err, "转换文件失败")
	}
	// submitConvertTask 之后 r.filePath 已经被更新为转换后的文件路径。
	info, err := os.Stat(r.filePath)
	if err != nil {
		g.Log().Criticalf(r.ctx, "获取转换后的文件信息失败: %v。文件地址：%s。", err, r.filePath)
		return nil, gerror.Wrap(err, "获取转换后的文件信息失败")
	}

	return &RecordingResult{
		ConnectID: gctx.CtxId(r.ctx),
		FilePath:  r.filePath,
		Dir:       r.dir,
		Size:      info.Size(),
		StartedAt: r.startTime,
		EndedAt:   time.Now(),
	}, nil
}

// Discard 关闭并删除当前录音。
// 备注：现在暂时没有用到这个功能。
func (r *Recorder) Discard() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fileIO.Close(); err != nil {
		g.Log().Errorf(r.ctx, "关闭文件流时发生错误。%v", err)
	}
	r.fileIO = nil
	if r.filePath != "" {
		if err := os.Remove(r.filePath); err != nil {
			g.Log().Errorf(r.ctx, "删除文件时发生错误。%v", err)
		}
	}
}

func writeWAVHeader(w io.Writer, channels, sampleRate, bitsPerSample int, dataSize int64) error {
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	header := struct {
		ChunkID       [4]byte
		ChunkSize     uint32
		Format        [4]byte
		Subchunk1ID   [4]byte
		Subchunk1Size uint32
		AudioFormat   uint16
		NumChannels   uint16
		SampleRate    uint32
		ByteRate      uint32
		BlockAlign    uint16
		BitsPerSample uint16
		Subchunk2ID   [4]byte
		Subchunk2Size uint32
	}{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     uint32(36 + dataSize),
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 16,
		AudioFormat:   1,
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		ByteRate:      uint32(byteRate),
		BlockAlign:    uint16(blockAlign),
		BitsPerSample: uint16(bitsPerSample),
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: uint32(dataSize),
	}

	return binary.Write(w, binary.LittleEndian, header)
}

func (r *Recorder) convertToWAV() error {
	// 打开源文件和目标文件
	src, err := os.Open(r.filePath)
	defer func() { _ = src.Close() }()
	if err != nil {
		return gerror.Wrap(err, "打开源文件失败")
	}
	dst, err := os.Create(r.filePath + ".wav")
	defer func() { _ = dst.Close() }()
	if err != nil {
		return gerror.Wrap(err, "创建目标文件失败")
	}

	// PCM -> WAV
	if err := writeWAVHeader(dst, r.channels, r.sampleRate, r.bitsPerSample, r.total); err != nil {
		return gerror.Wrap(err, "写入WAV头失败")
	}
	if _, err := io.Copy(dst, src); err != nil {
		return gerror.Wrap(err, "写入WAV数据失败")
	}

	if err := os.Remove(r.filePath); err != nil {
		return gerror.Wrap(err, "删除源文件失败")
	}
	// 把新文件的名字更新进去
	r.filePath = dst.Name()
	return nil
}
