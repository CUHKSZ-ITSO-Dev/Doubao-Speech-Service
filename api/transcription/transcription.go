// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package transcription

import (
	"context"

	"doubao-speech-service/api/transcription/v1"
)

type ITranscriptionV1 interface {
	Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error)
}
