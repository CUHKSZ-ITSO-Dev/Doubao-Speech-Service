package middlewares

import (
	"bytes"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// 自己写了一个 brotli 压缩中间件
func BrotliMiddleware(r *ghttp.Request) {
	// 1. 检查客户端是否支持 Brotli
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if !strings.Contains(acceptEncoding, "br") {
		// 不支持，则直接进入下一个处理流程
		r.Middleware.Next()
		return
	}

	// 2. 先执行业务逻辑
	r.Middleware.Next()

	// 3. 业务逻辑执行完毕后，获取响应内容
	// 注意: 只有当响应状态码为 200 且响应内容不为空时才进行压缩
	if r.Response.Status != 200 || r.Response.BufferLength() == 0 {
		return
	}

	// 4. 对响应内容进行 Brotli 压缩
	originalBody := r.Response.Buffer()
	var compressedBody bytes.Buffer
	
    // Brotli 提供了不同的压缩级别，这里使用级别11
	writer := brotli.NewWriterLevel(&compressedBody, 11)
	_, err := writer.Write(originalBody)
	if err != nil {
		g.Log().Errorf(r.Context(), "Brotli 写入失败: %v", err)
		return
	}
	err = writer.Close()
    if err != nil {
        g.Log().Errorf(r.Context(), "Brotli 写入器关闭失败: %v", err)
		return
    }

	// 5. 设置响应头，并用压缩后的内容替换原始响应
	r.Response.Header().Set("Content-Encoding", "br")
	// Vary 头告诉代理服务器，响应内容根据 Accept-Encoding 的不同而不同
	r.Response.Header().Set("Vary", "Accept-Encoding") 
	r.Response.ClearBuffer() // 清空原始未压缩的 buffer
	r.Response.Write(compressedBody.Bytes())
}