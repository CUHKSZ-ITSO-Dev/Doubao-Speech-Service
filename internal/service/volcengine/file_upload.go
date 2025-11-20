package volcengine

import (
	"context"
	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

type FileUploadResult struct {
	FileInfo v1.FileInfo
	FileName string
	Error    error
}

// UploadSource 抽象上传文件来源，便于复用上传逻辑。
type UploadSource interface {
	FileName() string
	FileSize() int64
	Open() (multipart.File, error)
}

// ProcessFileUpload 处理单个文件的上传
func ProcessFileUpload(ctx context.Context, file UploadSource, upn string) FileUploadResult {
	result := FileUploadResult{
		FileName: file.FileName(),
	}
	if file.FileSize() >= consts.MaxUploadSize {
		result.Error = gerror.Newf("文件大小超过最大限制：%d / 1,073,741,824 字节", file.FileSize())
		return result
	}

	// 打开文件
	fileReader, err := file.Open()
	if err != nil {
		result.Error = gerror.Wrap(err, "打开文件失败")
		return result
	}
	defer fileReader.Close()

	// 生成文件ID和验证文件类型
	mType, err := mimetype.DetectReader(fileReader)
	if err != nil {
		result.Error = gerror.Wrap(err, "检测文件类型失败")
		return result
	}
	_, ok := consts.TranscriptionExt[mType.Extension()]
	if !ok {
		result.Error = gerror.Newf("不支持的文件格式：%s", mType.Extension())
		return result
	}

	// 重置文件读取器，因为 mimetype.DetectReader 已经读取了一部分
	if _, err := fileReader.Seek(0, 0); err != nil {
		result.Error = gerror.Wrap(err, "无法重置文件读取器")
		return result
	}

	// 文件校验成功，进入 pending 状态
	fileID := uuid.New().String()
	if _, err := dao.Transcription.Ctx(ctx).Data(g.Map{
		"request_id": fileID,
		"owner":      upn,
		"file_info": g.Map{
			"object_key": fileID + "/" + file.FileName(),
			"filename":   file.FileName(),
			"file_type":  mType.Extension(),
			"file_size":  file.FileSize(),
		},
		"status": "pending",
	}).Insert(); err != nil {
		result.Error = gerror.Wrap(err, "数据库新建记录失败")
		return result
	}

	// 上传到TOS
	tosC := GetClient()
	key := fileID + "/" + file.FileName()
	if _, err = tosC.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
			Key:    key,
		},
		Content: fileReader,
	}); err != nil {
		result.Error = gerror.Wrap(err, "上传文件失败")
		return result
	}
	// 获取预签名URL
	url, err := tosC.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
		Key:        key,
	})
	if err != nil {
		result.Error = gerror.Wrap(err, "获取文件访问地址失败")
		return result
	}

	// 保存文件记录到数据库
	if _, err = dao.Transcription.Ctx(ctx).Data(g.Map{
		"task_params": g.Map{
			"Input": g.Map{
				"Offline": g.Map{
					"FileURL":  url.SignedUrl,
					"FileType": consts.TranscriptionExt[mType.Extension()],
				},
			},
		},
		"status": "uploaded", // 文件已上传，但任务未提交
	}).Where("request_id = ?", fileID).Update(); err != nil {
		result.Error = gerror.Wrap(err, "数据库写入 TOS 下载地址失败")
		return result
	}

	result.FileInfo = v1.FileInfo{
		FileID:   fileID,
		FileURL:  url.SignedUrl,
		FileType: mType.Extension(),
		FileSize: file.FileSize(),
		FileName: file.FileName(),
	}

	return result
}

// 从 HTTP 请求中获取上传文件
type HttpUploadSource struct {
	file *ghttp.UploadFile
}

// NewHttpUploadSource creates a new HttpUploadSource from an UploadFile
func NewHttpUploadSource(file *ghttp.UploadFile) *HttpUploadSource {
	return &HttpUploadSource{file: file}
}

func (h *HttpUploadSource) FileName() string {
	return h.file.Filename
}

func (h *HttpUploadSource) FileSize() int64 {
	return h.file.Size
}

func (h *HttpUploadSource) Open() (multipart.File, error) {
	return h.file.Open()
}

// 从本地文件中获取上传文件
type localUploadFile struct {
	path string
	size int64
}

func NewLocalUploadFile(path string) (*localUploadFile, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &localUploadFile{path: path, size: fileInfo.Size()}, nil
}

func (r *localUploadFile) FileName() string {
	return filepath.Base(r.path)
}

func (r *localUploadFile) FileSize() int64 {
	return r.size
}

func (r *localUploadFile) Open() (multipart.File, error) {
	return os.Open(r.path)
}
