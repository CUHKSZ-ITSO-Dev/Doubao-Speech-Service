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

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// 转换选项
type ConvertOptions struct {
	TargetFormat string   // e.g. "ogg", "mp3"。要求纯小写，前面不要点
	AudioBitrate string   // e.g. "64k"
	ExtraArgs    []string // appended raw ffmpeg arguments
	DeleteInput  bool     // remove the source file after conversion
}

// FFmpeg 转换器
type FFmpegConverter struct {
	binPath string
	opts    ConvertOptions
}

func NewConverter(binPath string, opts ConvertOptions) *FFmpegConverter {
	return &FFmpegConverter{
		binPath: binPath,
		opts:    opts,
	}
}

// 使用 ffmpeg 将输入文件转换为目标格式。根据文件的后缀名自动判断输入文件的格式，然后转换为目标格式。
//
// 参数:
//   - inputPath: string - 输入文件的路径
//   - extraArgs: g.Map - 转换的额外参数。原文件为 PCM 时需要提供一些参数。
//
// 返回:
//   - outputPath: string - 输出文件的路径
//   - err: error - 转换过程中发生的任何错误
func (c *FFmpegConverter) Convert(ctx context.Context, inputPath string, extraArgs g.MapStrStr) (outputPath string, err error) {
	if _, err := os.Stat(inputPath); err != nil {
		return "", gerror.Wrap(err, "输入文件不可访问")
	}

	target := fmt.Sprintf("%s.%s", strings.TrimSuffix(inputPath, filepath.Ext(inputPath)), c.opts.TargetFormat)
	args := []string{"-y"}
	// 对于 PCM 原始音频流需要做特殊处理
	if filepath.Ext(inputPath) == ".pcm" {
		if sampleRate, ok := extraArgs["ar"]; ok {
			args = append(args, "-ar", sampleRate)
			delete(extraArgs, "ar")
		} else {
			return "", gerror.New("covert from PCM: sample rate is required")
		}
		if channels, ok := extraArgs["ac"]; ok {
			args = append(args, "-ac", channels)
			delete(extraArgs, "ac")
		} else {
			return "", gerror.New("covert from PCM: channels is required")
		}
		if format, ok := extraArgs["f"]; ok {
			args = append(args, "-f", format)
			delete(extraArgs, "f")
		} else {
			return "", gerror.New("covert from PCM: format is required")
		}
	}
	args = append(args, "-i", inputPath, "-vn")
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
		return "", gerror.Wrapf(err, "ffmpeg convert to %s failed: %s", c.opts.TargetFormat, strings.TrimSpace(stderr.String()))
	}

	if c.opts.DeleteInput {
		err := os.Remove(inputPath)
		if err != nil {
			// 至少日志告个警
			g.Log().Errorf(ctx, "remove input file failed: %v", err)
		}
	}
	return target, nil
}
