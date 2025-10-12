package volcengine

import (
	"context"
	"fmt"
	"io"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

func TosUpload(ctx context.Context, objectKey string, file io.Reader) error {
	output, err := client.PutObjectV2(ctx, &tos.PutObjectV2Input{
	   PutObjectBasicInput: tos.PutObjectBasicInput{
		  Bucket: g.Cfg().MustGet(ctx, "volc.tos.bucket").String(),
		  Key:    objectKey,
	   },
	   Content: file,
	})
	if err != nil {
		if serverErr, ok := err.(*tos.TosServerError); ok {
			fmt.Println("Error:", serverErr.Error())
			fmt.Println("Request ID:", serverErr.RequestID)
			fmt.Println("Response Status Code:", serverErr.StatusCode)
			fmt.Println("Response Header:", serverErr.Header)
			fmt.Println("Response Err Code:", serverErr.Code)
			fmt.Println("Response Err Msg:", serverErr.Message)
			return gerror.Wrap(serverErr, "TOS 上传失败")
		} else {
			fmt.Println("Error:", err)
			return gerror.Wrap(err, "TOS 上传失败")
		}
	}
	fmt.Println("TOS 上传成功", output.RequestID)
	return nil
}
