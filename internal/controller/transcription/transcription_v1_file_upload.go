package transcription

import (
	"context"
	"path/filepath"
	"strings"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

// FileUpload 文件上传接口
func (c *ControllerV1) FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error) {
	// 获取上传的文件
	file := g.RequestFromCtx(ctx).GetUploadFile("file")
	if file == nil {
		return nil, gerror.New("上传文件为空，请使用字段名'file'上传文件")
	}

	// 生成文件ID和验证文件类型
	fileID := uuid.New().String()
	fileType := c.getFileType(file.Filename)
	if fileType == "" {
		return nil, gerror.New("不支持的文件格式，仅支持音频和视频文件")
	}

	// 打开文件
	fileReader, err := file.Open()
	if err != nil {
		return nil, gerror.Wrap(err, "打开文件失败")
	}
	defer fileReader.Close()

	// 上传到TOS
	tosC := volcengine.GetClient()
	key := fileID + "/" + file.Filename
	if _, err = tosC.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
			Key:    key,
		},
		Content: fileReader,
	}); err != nil {
		return nil, gerror.Wrap(err, "上传文件失败")
	}

	// 获取预签名URL
	url, err := tosC.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     g.Cfg().MustGet(ctx, "volc.lark.tos.bucket").String(),
		Key:        key,
	})
	if err != nil {
		return nil, gerror.Wrap(err, "获取文件访问地址失败")
	}

	// 保存文件记录到数据库
	_, err = dao.Transcription.Ctx(ctx).Data(g.Map{
		"request_id": fileID,
		"filename":   file.Filename,
		"file_url":   url.SignedUrl,
		"file_type":  fileType,
		"file_size":  file.Size,
		"status":     "uploaded", // 文件已上传，但任务未提交
	}).Insert()
	if err != nil {
		return nil, gerror.Wrap(err, "保存文件记录失败")
	}

	return &v1.FileUploadRes{
		FileID:   fileID,
		FileURL:  url.SignedUrl,
		FileType: fileType,
		FileSize: file.Size,
		FileName: file.Filename,
	}, nil
}

// getFileType 根据文件扩展名判断文件类型
func (c *ControllerV1) getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// 音频格式
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".m4a", ".ogg", ".wma"}
	for _, audioExt := range audioExts {
		if ext == audioExt {
			return "audio"
		}
	}

	// 视频格式
	videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}
	for _, videoExt := range videoExts {
		if ext == videoExt {
			return "video"
		}
	}

	return ""
}
