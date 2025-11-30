package transcription

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
	meetingRecordSvc "doubao-speech-service/internal/service/meetingRecord"
)

func (c *ControllerV1) UploadFile(ctx context.Context, req *v1.UploadFileReq) (res *v1.UploadFileRes, err error) {
	// 获取上传的文件
	uploadFiles := g.RequestFromCtx(ctx).GetUploadFiles("files")
	if uploadFiles == nil {
		return nil, gerror.New("上传文件为空，请使用字段名'files'上传文件")
	}

	userID := g.RequestFromCtx(ctx).Header.Get("X-User-ID")
	uploadDir := g.Cfg().MustGet(ctx, "meeting.record.dir", "/app/uploads").String()

	// 创建存储目录
	fileDir := filepath.Join(uploadDir, time.Now().Format("2006_01_02"), gctx.CtxId(ctx))
	if err := os.MkdirAll(fileDir, 0o755); err != nil {
		return nil, gerror.Wrap(err, "创建上传目录失败")
	}

	var successTaskMetas []v1.TaskMeta
	var errorFiles []v1.FileError

	// 处理每个文件：保存到本地 → 创建 pending 记录 → 提交到队列
	// 文件上传时：
	// - 保存的目录：Trace ID
	// - 文件名：文件索引-文件名。文件索引是 0,1,2,...
	// - requestID：Trace ID-文件索引
	// - TOS Object Key: Trace ID-文件索引/文件名
	for id, file := range uploadFiles {
		requestID := fmt.Sprintf("%s-%d", gctx.CtxId(ctx), id)
		fileName := fmt.Sprintf("%d-%s", id, file.Filename)
		localPath := filepath.Join(fileDir, fileName)

		// 1. 保存文件到本地
		if _, err := file.Save(localPath, true); err != nil {
			errorFiles = append(errorFiles, v1.FileError{
				FileName: fileName,
				Error:    "保存文件失败: " + err.Error(),
			})
			continue
		}

		// 2. 创建 pending 记录
		if _, err := dao.Transcription.Ctx(ctx).Data(g.Map{
			"request_id": requestID,
			"owner":      userID,
			"file_info": g.Map{
				"object_key": fmt.Sprintf("%s/%s", requestID, fileName),
				"filename":   file.Filename,
				"file_type":  "Pending Inspection", // 待检测
				"file_size":  file.Size,
			},
			"status": "pending",
		}).Insert(); err != nil {
			errorFiles = append(errorFiles, v1.FileError{
				FileName: file.Filename,
				Error:    "创建数据库记录失败: " + err.Error(),
			})
			// 创建数据库记录失败，删除已保存的文件（还未提交到队列）
			_ = os.Remove(localPath)
			continue
		}

		// 3. 提交到上传队列（异步处理）
		// 注意：一旦提交到队列，文件就不能在这里删除了！
		// uploadWorker 会在上传完成后自动删除文件
		meetingRecordSvc.EnqueueUpload(ctx, &meetingRecordSvc.RecordingResult{
			ConnectID: requestID,
			Owner:     userID,
			FilePath:  localPath,
			Dir:       fileDir,
			Size:      file.Size,
			StartedAt: time.Now(),
			EndedAt:   time.Now(),
		})

		// 4. 立即返回 TaskMeta（不等待上传完成）
		successTaskMetas = append(successTaskMetas, v1.TaskMeta{
			RequestId: requestID,
			Owner:     userID,
			Status:    "pending",
			CreatedAt: nil,
		})
	}

	return &v1.UploadFileRes{
		TaskMetas: successTaskMetas,
		Errors:    errorFiles,
		Total:     len(uploadFiles),
		Success:   len(successTaskMetas),
		Failed:    len(errorFiles),
	}, nil
}
