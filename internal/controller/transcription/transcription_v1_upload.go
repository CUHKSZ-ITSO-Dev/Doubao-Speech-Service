package transcription

import (
	"context"
	"time"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/transcription"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/google/uuid"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	requestId := uuid.New().String()
	gClient := g.Client()

	files := g.RequestFromCtx(ctx).GetUploadFiles("transcription")
	if files == nil {
		return nil, gerror.New("上传文件为空")
	}

	for _, file := range files {
		fileReader, err := file.Open()
		if err != nil {
			return nil, gerror.Wrap(err, "打开文件失败")
		}
		defer fileReader.Close()
		err = volcengine.TosUpload(ctx, requestId+"/"+file.Filename, fileReader)
		if err != nil {
			return nil, gerror.Wrap(err, "上传文件失败")
		}

		// 获取 S3 存储桶文件下载地址
		tosC := volcengine.GetClient()
		url, err := tosC.PreSignedURL(&tos.PreSignedURLInput{
			HTTPMethod: enum.HttpMethodGet,
			Bucket:     g.Cfg().MustGet(ctx, "volc.lark.tos.bucket").String(),
			Key:        requestId + "/" + file.Filename,
		})
		if err != nil {
			return nil, gerror.Wrap(err, "获取 S3 存储桶文件下载地址失败")
		}
		req.Input.Offline.FileURL = url.SignedUrl

		client, err := gClient.ContentJson().
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
			bgCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()

			for range t.C {
				status, err := transcription.Query(bgCtx, res.Data.TaskID, requestId)
				if err != nil || status == "running" || status == "pending" {
					continue
				}
				break
			}
		}()
	}
	return
}
