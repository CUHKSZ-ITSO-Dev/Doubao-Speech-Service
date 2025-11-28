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
	ctx       context.Context
	connectID string

	mu            sync.Mutex
	file          *os.File
	dir           string
	filePath      string
	wavPath       string
	total         int64
	closed        bool
	maxBytes      int64
	startTime     time.Time
	sampleRate    int
	bitsPerSample int
	channels      int
	converter     *media.FFmpegConverter
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

func NewRecorder(ctx context.Context, connectID string) (*Recorder, error) {
	opts := getOptions()
	if opts.Dir == "" {
		return nil, ErrRecorderDisabled
	}
	now := time.Now()
	formattedDate := now.Format("20060102")
	formattedTime := now.Format("20060102150405")
	dir := filepath.Join(opts.Dir, formattedDate, connectID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, gerror.Wrap(err, "创建目录失败")
	}
	fileName := "Meeting_" + formattedTime
	path := filepath.Join(dir, fileName+".pcm")
	file, err := os.Create(path)
	if err != nil {
		return nil, gerror.Wrap(err, "创建文件失败")
	}

	return &Recorder{
		ctx:           ctx,
		connectID:     connectID,
		file:          file,
		dir:           dir,
		filePath:      path,
		wavPath:       filepath.Join(dir, fileName+".wav"),
		startTime:     now,
		maxBytes:      opts.MaxBytes,
		sampleRate:    opts.SampleRate,
		bitsPerSample: opts.BitsPerSample,
		channels:      opts.Channels,
		converter:     getConverter(),
	}, nil
}

func (r *Recorder) ConnectID() string {
	if r == nil {
		return ""
	}
	return r.connectID
}

// Append 写入一帧音频。若超过限制或写入失败，会终止录制。
func (r *Recorder) Append(frame []byte) error {
	if r == nil {
		return gerror.New("录音器被禁用")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.file == nil {
		return gerror.New("录音器已关闭")
	}
	if r.maxBytes > 0 && r.total+int64(len(frame)) > r.maxBytes {
		r.closeLocked()
		return gerror.New("录音器已关闭")
	}
	n, err := r.file.Write(frame)
	if err != nil {
		r.closeLocked()
		return gerror.Wrap(err, "写入文件失败")
	}
	r.total += int64(n)
	return nil
}

// Finalize 结束录制。返回 nil 表示没有有效数据。
func (r *Recorder) Finalize() (*RecordingResult, error) {
	if r == nil {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return nil, gerror.Wrap(err, "关闭文件失败")
		}
		r.file = nil
	}
	r.closed = true
	if r.total == 0 {
		_ = os.Remove(r.filePath)
		return nil, nil
	}
	if err := r.flushToWAV(); err != nil {
		return nil, gerror.Wrap(err, "刷新到WAV文件失败")
	}

	finalPath := r.wavPath
	if r.converter != nil {
		// 提交转换任务到队列，等待结果
		resultCh := submitConvertTask(r.ctx, r.wavPath)
		result := <-resultCh
		if result.err != nil {
			return nil, gerror.Wrap(result.err, "转换文件失败")
		}
		finalPath = result.path
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		g.Log().Criticalf(r.ctx, "获取转换后的文件信息失败: %v。原始地址：%s。目标地址：%s。", err, r.wavPath, finalPath)
		return nil, gerror.Wrap(err, "获取文件信息失败")
	}

	return &RecordingResult{
		ConnectID: r.connectID,
		FilePath:  finalPath,
		Dir:       r.dir,
		Size:      info.Size(),
		StartedAt: r.startTime,
		EndedAt:   time.Now(),
	}, nil
}

// Discard 关闭并删除当前录音。
func (r *Recorder) Discard() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
	if r.filePath != "" {
		_ = os.Remove(r.filePath)
	}
}

func (r *Recorder) closeLocked() {
	if r.closed {
		return
	}
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	r.closed = true
}

func (r *Recorder) flushToWAV() error {
	src, err := os.Open(r.filePath)
	if err != nil {
		return gerror.Wrap(err, "打开源文件失败")
	}
	defer src.Close()

	dst, err := os.Create(r.wavPath)
	if err != nil {
		return gerror.Wrap(err, "创建目标文件失败")
	}

	if err := writeWAVHeader(dst, r.channels, r.sampleRate, r.bitsPerSample, r.total); err != nil {
		dst.Close()
		return gerror.Wrap(err, "写入WAV头失败")
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return gerror.Wrap(err, "写入WAV数据失败")
	}

	if err := dst.Close(); err != nil {
		return gerror.Wrap(err, "关闭目标文件失败")
	}
	return os.Remove(r.filePath)
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
