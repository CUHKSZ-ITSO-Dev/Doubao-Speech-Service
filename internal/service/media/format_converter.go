package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FormatConverter interface {
	Convert(ctx context.Context, inputPath string) (string, error)
}

// 转换选项
type ConvertOptions struct {
	TargetFormat string   // e.g. "ogg", "mp3"
	AudioBitrate string   // e.g. "64k"
	ExtraArgs    []string // appended raw ffmpeg arguments
	DeleteInput  bool     // remove the source file after conversion
}

// FFmpeg 转换器
type FFmpegConverter struct {
	binPath string
	opts    ConvertOptions
}

// 根据提供的 ffmpeg 路径和转换选项构建 FFmpegConverter
func NewFFmpegConverter(binPath string, opts ConvertOptions) (*FFmpegConverter, error) {
	format := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(opts.TargetFormat), "."))
	if format == "" {
		return nil, nil
	}
	opts.TargetFormat = format

	if binPath == "" {
		p, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
		}
		binPath = p
	}

	return &FFmpegConverter{
		binPath: binPath,
		opts:    opts,
	}, nil
}

// 使用 ffmpeg 将输入文件转换为目标格式
func (c *FFmpegConverter) Convert(ctx context.Context, inputPath string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return inputPath, nil
	}
	if _, err := os.Stat(inputPath); err != nil {
		return "", fmt.Errorf("input file not accessible: %w", err)
	}

	target := buildTargetPath(inputPath, c.opts.TargetFormat)
	args := []string{"-y", "-i", inputPath, "-vn"}
	if c.opts.AudioBitrate != "" {
		args = append(args, "-b:a", c.opts.AudioBitrate)
	}
	if len(c.opts.ExtraArgs) > 0 {
		args = append(args, c.opts.ExtraArgs...)
	}
	args = append(args, target)

	cmd := exec.CommandContext(ctx, c.binPath, args...)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			err = fmt.Errorf("%w: %s", err, detail)
		}
		return "", fmt.Errorf("ffmpeg convert to %s failed: %w", c.opts.TargetFormat, err)
	}

	if c.opts.DeleteInput {
		_ = os.Remove(inputPath)
	}
	return target, nil
}

func buildTargetPath(inputPath, format string) string {
	format = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	if format == "" {
		return inputPath
	}
	base := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))
	return fmt.Sprintf("%s.%s", base, format)
}
