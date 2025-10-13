package transcription

import (
	"context"
	"time"

	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
)

func Recover(ctx context.Context) {
	transRecords := []entity.Transcription{}
	dao.Transcription.Ctx(ctx).Where("status = ?", "pending").Scan(&transRecords)
	for _, v := range transRecords {
		go func(taskId, requestId string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()
			for range t.C {
				Query(bgCtx, taskId, requestId)
			}
		}(v.TaskId, v.RequestId)
	}
	if len(transRecords) > 0 {
		g.Log().Infof(ctx, "已恢复 pending 状态任务 %d 个", len(transRecords))
	}
}
