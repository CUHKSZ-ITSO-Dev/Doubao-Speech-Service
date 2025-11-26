// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package transcription

import (
	"context"

	"doubao-speech-service/api/transcription/v1"
)

type ITranscriptionV1 interface {
	UploadFile(ctx context.Context, req *v1.UploadFileReq) (res *v1.UploadFileRes, err error)
	TaskSubmit(ctx context.Context, req *v1.TaskSubmitReq) (res *v1.TaskSubmitRes, err error)
	GetTaskList(ctx context.Context, req *v1.GetTaskListReq) (res *v1.GetTaskListRes, err error)
	Search(ctx context.Context, req *v1.SearchReq) (res *v1.SearchRes, err error)
	GetTask(ctx context.Context, req *v1.GetTaskReq) (res *v1.GetTaskRes, err error)
	DeleteTask(ctx context.Context, req *v1.DeleteTaskReq) (res *v1.DeleteTaskRes, err error)
}
