package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"

	"doubao-speech-service/api/transcription/v1"
)

func (c *ControllerV1) DeleteTask(ctx context.Context, req *v1.DeleteTaskReq) (res *v1.DeleteTaskRes, err error) {
	return nil, gerror.NewCode(gcode.CodeNotImplemented)
}
