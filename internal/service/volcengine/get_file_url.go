package volcengine

import (
	"context"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

type FileURL struct {
	URL string `json:"url"`
}

// 根据任务记录获取文件直链地址
func GetFileURL(ctx context.Context, transRecord entity.Transcription) (FileURL, error) {
	tosC := GetClient()

	requestId := transRecord.RequestId
	fileName := transRecord.FileInfo.Get("filename").String()
	key := requestId + "/" + fileName
	url, err := tosC.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
		Key:        key,
		Expires:    3600,
	})
	if err != nil {
		return FileURL{}, gerror.Wrap(err, "获取文件访问地址失败")
	}
	return FileURL{
		URL: url.SignedUrl,
	}, nil
}
