package transcription

import (
	"context"

	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
)

func Recover(ctx context.Context) {
	transRecords := []entity.Transcription{}
	dao.Transcription.Ctx(ctx).Where("status = ?", "pending").Scan(&transRecords)
	for _, v := range transRecords {
		go Query(ctx, v.TaskId, v.RequestId)
	}
	g.Log().Criticalf(ctx, "已恢复 pending 状态任务 %d 个", len(transRecords))
}
