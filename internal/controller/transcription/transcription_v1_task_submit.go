package transcription

import (
	"context"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/transcription"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
)

// TaskSubmit 任务提交接口
func (c *ControllerV1) TaskSubmit(ctx context.Context, req *v1.TaskSubmitReq) (res *v1.TaskSubmitRes, err error) {
	// 验证文件ID是否存在
	transcriptionRecord, err := dao.Transcription.Ctx(ctx).Where("request_id", req.RequestId).One()
	if err != nil {
		return nil, gerror.Wrap(err, "查询文件信息失败")
	}
	if transcriptionRecord.IsEmpty() {
		return nil, gerror.New("文件ID不存在，请先上传文件")
	}

	// 检查文件状态
	if transcriptionRecord["status"].String() != "uploaded" {
		return nil, gerror.Newf("文件状态异常，无法提交任务。当前状态：%s", transcriptionRecord["status"].String())
	}

	// 构建提交给第三方API的请求
	err = volcengine.UpdateFileURL(ctx, req.RequestId) // 先更新文件URL
	if err != nil {
		return nil, gerror.Wrap(err, "更新文件URL失败")
	}
	submitReq := struct {
		Input struct {
			Offline struct {
				FileURL  string `json:"FileURL"`
				FileType string `json:"FileType"`
			} `json:"Offline"`
		} `json:"Input"`
		Params v1.TaskSubmitParams `json:"Params"`
	}{}

	if err = transcriptionRecord["task_params"].Struct(&submitReq); err != nil {
		return nil, gerror.Wrap(err, "解析数据库任务参数失败")
	}
	submitReq.Params = req.Params

	// 提交任务到第三方API
	gClient := g.Client()
	response, err := gClient.ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc.lark.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc.lark.accessKey").String(),
			"X-Api-Resource-Id": g.Cfg().MustGet(ctx, "volc.lark.service").String(),
			"X-Api-Request-Id":  transcriptionRecord["request_id"].String(),
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/submit",
			submitReq,
		)
	if err != nil {
		if response != nil {
			response.RawDump()
		}
		return nil, gerror.Wrap(err, "提交任务失败，POST 请求发生错误")
	}
	defer response.Close()

	// 解析响应
	bodyStr := response.ReadAllString()
	if response.Response.Header.Get("X-Api-Message") != "OK" {
		statusCode := response.Response.Header.Get("X-Api-Status-Code")
		logid := response.Response.Header.Get("X-Tt-Logid")
		bodyPreview := bodyStr
		if len(bodyPreview) > 500 {
			bodyPreview = gstr.SubStr(bodyPreview, 0, 500) + "..."
		}
		g.Log().Errorf(ctx, "[%s] 任务提交失败。StatusCode=%s Message=%s Mapped=%s Logid=%s Body=%s",
			transcriptionRecord["request_id"].String(),
			statusCode,
			response.Response.Header.Get("X-Api-Message"),
			consts.GetErrMsg(ctx, statusCode),
			logid,
			bodyPreview,
		)
		return nil, gerror.Newf("第三方服务返回非OK。StatusCode=%s Message=%s Logid=%s",
			statusCode,
			response.Response.Header.Get("X-Api-Message"),
			logid,
		)
	}

	taskID := gjson.New(bodyStr).Get("Data.TaskID").String()

	// 更新数据库记录
	_, err = dao.Transcription.Ctx(ctx).
		Where("request_id", req.RequestId).
		Data(g.Map{
			"task_id":     taskID,
			"task_params": submitReq,
			"status":      "submitted",
		}).Update()
	if err != nil {
		return nil, gerror.Wrap(err, "更新任务记录失败")
	}

	transcription.Polling(taskID, transcriptionRecord["request_id"].String())

	return &v1.TaskSubmitRes{
		Status: "pending",
	}, nil
}
