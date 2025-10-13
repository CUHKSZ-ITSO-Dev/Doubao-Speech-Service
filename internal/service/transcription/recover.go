package transcription

import (
	"context"

	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
)

func Recover(ctx context.Context) {
	transRecords := []entity.Transcription{}
	dao.Transcription.Ctx(ctx).WhereIn("status", []string{"pending", "running"}).Scan(&transRecords)
	for _, v := range transRecords {
		Polling(v.TaskId, v.RequestId)
	}
	if len(transRecords) > 0 {
		g.Log().Infof(ctx, "已恢复 pending 状态任务 %d 个", len(transRecords))
	}
}
