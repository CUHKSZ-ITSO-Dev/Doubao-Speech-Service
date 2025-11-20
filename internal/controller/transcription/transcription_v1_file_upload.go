package transcription

import (
	"context"
	"sync"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// FileUpload 文件上传接口（支持单文件和多文件）
func (c *ControllerV1) FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error) {
	// 获取上传的文件 - 支持多种方式
	uploadFiles := g.RequestFromCtx(ctx).GetUploadFiles("files")
	if uploadFiles == nil {
		return nil, gerror.New("上传文件为空，请使用字段名'files'上传文件")
	}

	// 并发处理多个文件
	var wg sync.WaitGroup
	resultCh := make(chan volcengine.FileUploadResult, len(uploadFiles))
	semaphore := make(chan struct{}, 3) // 限制并发数量

	userID := g.RequestFromCtx(ctx).Header.Get("X-User-ID")
	for _, file := range uploadFiles {
		wg.Add(1)
		go func(file *ghttp.UploadFile) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			resultCh <- volcengine.ProcessFileUpload(ctx, volcengine.NewHttpUploadSource(file), userID)
		}(file)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集处理结果
	var successFiles []v1.FileInfo
	var errorFiles []v1.FileError

	for result := range resultCh {
		if result.Error != nil {
			errorFiles = append(errorFiles, v1.FileError{
				FileName: result.FileName,
				Error:    result.Error.Error(),
			})
		} else {
			successFiles = append(successFiles, result.FileInfo)
		}
	}

	return &v1.FileUploadRes{
		Files:   successFiles,
		Errors:  errorFiles,
		Total:   len(uploadFiles),
		Success: len(successFiles),
		Failed:  len(errorFiles),
	}, nil
}
