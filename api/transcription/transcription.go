// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package transcription

import (
	"context"

	"doubao-speech-service/api/transcription/v1"
)

type ITranscriptionV1 interface {
	FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error)
	TaskSubmit(ctx context.Context, req *v1.TaskSubmitReq) (res *v1.TaskSubmitRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
}
