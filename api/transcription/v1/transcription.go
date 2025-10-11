package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

type UploadReq struct {
	g.Meta `path:"/upload" tags:"Upload" method:"post" summary:"Upload a file"`
}
type UploadRes struct {
	g.Meta `mime:"text/html" example:"string"`
}
