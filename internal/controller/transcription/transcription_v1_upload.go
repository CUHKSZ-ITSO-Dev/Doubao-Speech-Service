package transcription

import (
	"context"
	"sync"
	"time"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/service/transcription"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

type FileProcessResult struct {
	Filename  string
	TaskID    string
	RequestID string
	Error     error
}

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	files := g.RequestFromCtx(ctx).GetUploadFiles("transcription")
	if files == nil {
		return nil, gerror.New("上传文件为空")
	}

	// 并发处理多个文件
	var wg sync.WaitGroup
	resultCh := make(chan FileProcessResult, len(files))
	semaphore := make(chan struct{}, 3) // 单个用户 3 并发限制
	for _, file := range files {
		wg.Add(1)
		go func(file *ghttp.UploadFile) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			resultCh <- c.processFile(ctx, file, *req)
		}(file)
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集处理结果
	results := map[string]FileProcessResult{}
	errors := []error{}
	for result := range resultCh {
		if result.Error != nil {
			errors = append(errors, gerror.Newf("文件 %s 处理失败: %v", result.Filename, result.Error))
		}
		results[result.Filename] = result
	}

	if len(results) == len(errors) {
		return nil, gerror.Newf("所有文件处理失败: %v", errors)
	}
	if len(errors) > 0 {
		g.Log().Warningf(ctx, "部分文件处理失败: %v", errors)
	}
	res = &v1.UploadRes{
		Result: gjson.New(results),
	}
	return
}

// processFile 处理单个文件的上传和提交
func (c *ControllerV1) processFile(ctx context.Context, file *ghttp.UploadFile, req v1.UploadReq) FileProcessResult {
	gClient := g.Client()
	requestId := uuid.New().String()
	result := FileProcessResult{
		Filename:  file.Filename,
		RequestID: requestId,
	}

	// 打开文件
	fileReader, err := file.Open()
	if err != nil {
		result.Error = gerror.Wrap(err, "打开文件失败")
		return result
	}
	defer fileReader.Close()

	// 上传到TOS，并获取文件下载地址
	tosC := volcengine.GetClient()
	if _, err = tosC.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
			Key:    requestId + "/" + file.Filename,
		},
		Content: fileReader,
	}); err != nil {
		result.Error = gerror.Wrap(err, "上传文件失败")
		return result
	}
	url, err := tosC.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     g.Cfg().MustGet(ctx, "volc.lark.tos.bucket").String(),
		Key:        requestId + "/" + file.Filename,
	})
	if err != nil {
		result.Error = gerror.Wrap(err, "获取 S3 存储桶文件下载地址失败")
		return result
	}
	req.Input.Offline.FileURL = url.SignedUrl

	// 提交任务到API
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
			req,
		)
	if err != nil {
		result.Error = gerror.Wrap(err, "提交任务失败，POST 请求发生错误")
		return result
	}
	defer response.Close()

	// 解析响应
	if response.Response.Header.Get("X-Api-Message") != "OK" {
		result.Error = gerror.Newf(
			"第三方服务通知任务处理失败。错误码：%s，错误信息：%s。Logid：%s",
			response.Response.Header.Get("X-Api-Error-Message"),
			consts.GetErrMsg(response.Response.Header.Get("X-Api-Error-Message")),
			response.Response.Header.Get("X-Tt-Logid"),
		)
		return result
	}
	result.TaskID = gjson.New(response.ReadAllString()).Get("Data.TaskID").String()

	// 写入数据库
	_, err = dao.Transcription.Ctx(ctx).Data(g.Map{
		"task_id":       result.TaskID,
		"request_id":    requestId,
		"upload_params": req,
		"status":        "pending",
		"filename":      file.Filename,
	}).Insert()
	if err != nil {
		result.Error = gerror.Wrap(err, "写入数据库失败")
		return result
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
	}(result.TaskID, requestId)

	return result
}
