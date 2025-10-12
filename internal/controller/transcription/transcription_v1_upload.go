package transcription

import (
	"context"
	"time"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/transcription"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/google/uuid"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	requestId := uuid.New().String()
	client, err := g.Client().ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc-lark-minutes.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc-lark-minutes.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  requestId,
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/submit",
			req,
		)
	if err != nil {
		return nil, gerror.Wrap(err, "提交任务失败，POST 请求发生错误")
	}
	defer client.Close()

	if client.Response.Header.Get("X-Api-Message") != "OK" {
		return nil, gerror.Newf(
			"第三方服务通知任务处理失败。错误码：%s，错误信息：%s。Logid：%s",
			client.Response.Header.Get("X-Api-Error-Message"),
			consts.GetErrMsg(client.Response.Header.Get("X-Api-Error-Message")),
			client.Response.Header.Get("X-Tt-Logid"),
		)
	}

	if err = gconv.Struct(client.ReadAllString(), &res); err != nil {
		return nil, gerror.Wrap(err, "返回结果格式化失败")
	}
	res.Data.RequestID = requestId

	_, err = dao.Transcription.Ctx(ctx).Data(g.Map{
		"task_id":       res.Data.TaskID,
		"request_id":    requestId,
		"upload_params": req,
		"status":        "pending",
	}).Insert()
	if err != nil {
		return nil, gerror.Wrap(err, "写入数据库失败")
	}

	go func() {
		bgCtx := context.Background()
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()

		for range t.C {
			status, err := transcription.Query(bgCtx, res.Data.TaskID, requestId)
			if err != nil {
				continue
			}
			if status != "running" {
				// TODO - 写入数据库
				return
			}
		}
	}()

	return
}
