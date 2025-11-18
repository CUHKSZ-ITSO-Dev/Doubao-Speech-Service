package meetingRecord

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gorilla/websocket"
)

func ProxyWebSocket(ctx context.Context, srcName string, src, dst *websocket.Conn, recorder *Recorder, errCh chan<- error) {
	logger := g.Log()
	recorderActive := recorder != nil && srcName == "client"
	var finalErr error
	defer func() {
		if errCh != nil {
			if finalErr == nil {
				finalErr = &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal"}
			}
			errCh <- finalErr
		}
	}()
	for {
		msgType, msg, err := src.ReadMessage()
		if err != nil {
			finalErr = err
			if closeErr, ok := err.(*websocket.CloseError); ok {
				logger.Infof(ctx, "%s 连接关闭，code=%d, text=%s", srcName, closeErr.Code, closeErr.Text)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(closeErr.Code, closeErr.Text),
					time.Now().Add(time.Second),
				)
				finalErr = closeErr
			} else if errors.Is(err, io.EOF) {
				logger.Infof(ctx, "%s 连接已结束", srcName)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "EOF"),
					time.Now().Add(time.Second),
				)
				finalErr = &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "EOF"}
			} else if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
				logger.Infof(ctx, "%s 连接已关闭", srcName)
				finalErr = &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "connection closed"}
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.Infof(ctx, "%s 上下文结束: %v", srcName, err)
				finalErr = &websocket.CloseError{Code: websocket.CloseGoingAway, Text: err.Error()}
			} else if websocket.IsUnexpectedCloseError(err) {
				logger.Warningf(ctx, "%s 连接异常关闭: %v", srcName, err)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "unexpected close"),
					time.Now().Add(time.Second),
				)
			}
			return
		}

		if recorderActive && msgType == websocket.BinaryMessage {
			pcm, handled, err := extractPCMFromFrame(msg)
			if err != nil {
				logger.Warningf(ctx, "%s 录音帧解析失败: %v", srcName, err)
				recorderActive = false
			} else if len(pcm) > 0 {
				if err := recorder.Append(pcm); err != nil {
					logger.Warningf(ctx, "%s 录音写入失败: %v", srcName, err)
					recorderActive = false
				}
			} else if handled {
				// 帧已解析但无需写入（例如 full client request），直接跳过。
			} else {
				logger.Debugf(ctx, "%s 收到非音频二进制帧，message_type 未处理", srcName)
			}
		}

		// if srcName == "upstream" {
		// 	// 处理上游消息的日志输出
		// 	var msgStr string
		// 	if msgType == websocket.TextMessage {
		// 		// 文本消息直接使用
		// 		msgStr = string(msg)
		// 	} else {
		// 		msgBytes := msg
		// 		jsonStart := -1
		// 		for i, b := range msgBytes {
		// 			if b == '{' {
		// 				jsonStart = i
		// 				break
		// 			}
		// 		}
		// 		if jsonStart >= 0 && jsonStart < len(msgBytes) {
		// 			msgStr = string(msgBytes[jsonStart:])
		// 		} else {
		// 			msgStr = string(msgBytes)
		// 		}
		// 	}
		// 	logger.Infof(ctx, msgStr)
		// }

		if err := dst.WriteMessage(msgType, msg); err != nil {
			finalErr = err
			return
		}
	}
}
