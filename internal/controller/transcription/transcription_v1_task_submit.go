package transcription

import (
	"context"
	"time"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/transcription"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

// TaskSubmit 任务提交接口
func (c *ControllerV1) TaskSubmit(ctx context.Context, req *v1.TaskSubmitReq) (res *v1.TaskSubmitRes, err error) {
	// 验证文件ID是否存在
	fileInfo, err := dao.Transcription.Ctx(ctx).Where("request_id", req.FileID).One()
	if err != nil {
		return nil, gerror.Wrap(err, "查询文件信息失败")
	}
	if fileInfo.IsEmpty() {
		return nil, gerror.New("文件ID不存在，请先上传文件")
	}

	// 检查文件状态
	if fileInfo["status"].String() != "uploaded" {
		return nil, gerror.New("文件状态异常，无法提交任务")
	}

	// 生成新的任务ID
	requestId := uuid.New().String()

	// 构建提交给第三方API的请求
	submitReq := struct {
		Input struct {
			Offline struct {
				FileURL  string `json:"FileURL"`
				FileType string `json:"FileType"`
			} `json:"Offline"`
		} `json:"Input"`
		Params v1.TaskSubmitReq `json:"Params"`
	}{}

	submitReq.Input.Offline.FileURL = fileInfo["file_url"].String()
	submitReq.Input.Offline.FileType = consts.TranscriptionExt[fileInfo["file_type"].String()]
	submitReq.Params = *req

	// 提交任务到第三方API
	gClient := g.Client()
	response, err := gClient.ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc.lark.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc.lark.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  requestId,
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/submit",
			submitReq,
		)
	if err != nil {
		return nil, gerror.Wrap(err, "提交任务失败，POST 请求发生错误")
	}
	defer response.Close()

	// 解析响应
	if response.Response.Header.Get("X-Api-Message") != "OK" {
		return nil, gerror.Newf(
			"第三方服务通知任务处理失败。错误码：%s，错误信息：%s。Logid：%s",
			response.Response.Header.Get("X-Api-Error-Message"),
			consts.GetErrMsg(response.Response.Header.Get("X-Api-Error-Message")),
			response.Response.Header.Get("X-Tt-Logid"),
		)
	}

	taskID := gjson.New(response.ReadAllString()).Get("Data.TaskID").String()

	// 更新数据库记录
	_, err = dao.Transcription.Ctx(ctx).
		Where("request_id", req.FileID).
		Data(g.Map{
			"task_id":       taskID,
			"upload_params": req,
			"status":        "pending",
		}).Update()
	if err != nil {
		return nil, gerror.Wrap(err, "更新任务记录失败")
	}

	// 启动后台任务轮询结果
	go func(taskID, reqID string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
		defer cancel()
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()

		for range t.C {
			status, err := transcription.Query(bgCtx, taskID, reqID)
			if err != nil || status == "running" || status == "pending" {
				continue
			}
			break
		}
	}(taskID, requestId)

	return &v1.TaskSubmitRes{
		TaskID:    taskID,
		RequestID: requestId,
		Status:    "pending",
	}, nil
}
