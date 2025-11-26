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
)

var (
	ErrRecorderDisabled = errors.New("recorder disabled")
	errRecorderClosed   = errors.New("recorder closed")
)

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
	formattedDate := now.Format("2006_01_02")
	formattedTime := now.Format("2006_01_02_150405")
	dir := filepath.Join(opts.Dir, formattedDate, connectID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	fileName := "Meeting_" + formattedTime
	path := filepath.Join(dir, fileName+".pcm")
	file, err := os.Create(path)
	if err != nil {
		return nil, err
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
		return ErrRecorderDisabled
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.file == nil {
		return errRecorderClosed
	}
	if r.maxBytes > 0 && r.total+int64(len(frame)) > r.maxBytes {
		r.closeLocked()
		return errRecorderClosed
	}
	n, err := r.file.Write(frame)
	if err != nil {
		r.closeLocked()
		return err
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
			return nil, err
		}
		r.file = nil
	}
	r.closed = true
	if r.total == 0 {
		_ = os.Remove(r.filePath)
		return nil, nil
	}
	if err := r.flushToWAV(); err != nil {
		return nil, err
	}

	return &RecordingResult{
		ConnectID: r.connectID,
		FilePath:  r.wavPath,
		Dir:       r.dir,
		Size:      r.total,
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
		return err
	}
	defer src.Close()

	dst, err := os.Create(r.wavPath)
	if err != nil {
		return err
	}

	if err := writeWAVHeader(dst, r.channels, r.sampleRate, r.bitsPerSample, r.total); err != nil {
		dst.Close()
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}

	if err := dst.Close(); err != nil {
		return err
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
