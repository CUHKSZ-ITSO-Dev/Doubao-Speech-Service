package volcengine

import (
	"context"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

// 根据任务记录获取文件直链地址
func GetFileURL(ctx context.Context, transRecord *entity.Transcription) (string, error) {
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
		return "", gerror.Wrap(err, "获取文件访问地址失败")
	}
	return url.SignedUrl, nil
}

// 更新指定 Transcription 记录的文件 URL
func UpdateFileURL(ctx context.Context, requestId string) error {
	transRecord := entity.Transcription{}
	if err := dao.Transcription.Ctx(ctx).Where("request_id = ?", requestId).Scan(&transRecord); err != nil {
		return gerror.Wrap(err, "查询任务记录失败")
	}
	fileURL, err := GetFileURL(ctx, &transRecord)
	if err != nil {
		return gerror.Wrap(err, "获取文件URL失败")
	}
	transRecord.TaskParams.Set("Input.Offline.FileURL", fileURL)
	g.Log().Infof(ctx, "更新文件URL: %s", fileURL)
	if _, err := dao.Transcription.Ctx(ctx).Data(transRecord).Where("request_id = ?", requestId).Update(); err != nil {
		return gerror.Wrap(err, "更新任务记录失败")
	}
	return nil
}
