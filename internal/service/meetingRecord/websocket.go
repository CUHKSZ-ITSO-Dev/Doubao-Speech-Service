package meetingRecord

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gorilla/websocket"
)

func ProxyWebSocket(ctx context.Context, srcName string, src, dst *websocket.Conn, recorder *Recorder, errCh chan<- error, taskCompleteCh <-chan *RecordingResult, serverFinalReceivedCh chan<- struct{}) {
	recorderActive := recorder != nil && srcName == "client"
	waitingForTaskComplete := false
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
		// 如果在等待任务完成，检查是否有任务完成的消息
		if waitingForTaskComplete && taskCompleteCh != nil {
			select {
			case result := <-taskCompleteCh:
				if result != nil {
					g.Log().Infof(ctx, "录音任务已完成，准备发送任务信息")
					if err := sendTaskCompleteMessage(ctx, dst, result); err != nil {
						g.Log().Warningf(ctx, "发送任务完成消息失败: %v", err)
					} else {
						g.Log().Infof(ctx, "任务完成消息已发送，等待客户端确认...")
					}
				}
				waitingForTaskComplete = false
			default:
				// 没有任务完成消息，继续处理
			}
		}

		// 设置读取超时，避免在等待任务完成时阻塞太久
		if waitingForTaskComplete {
			_ = src.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		} else {
			_ = src.SetReadDeadline(time.Time{}) // 清除超时
		}

		msgType, msg, err := src.ReadMessage()
		if err != nil {
			// 如果是超时错误且正在等待任务完成，继续循环
			if waitingForTaskComplete {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
			}

			finalErr = err
			if closeErr, ok := err.(*websocket.CloseError); ok {
				g.Log().Infof(ctx, "%s 连接关闭，code=%d, text=%s", srcName, closeErr.Code, closeErr.Text)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(closeErr.Code, closeErr.Text),
					time.Now().Add(time.Second),
				)
				finalErr = closeErr
			} else if errors.Is(err, io.EOF) {
				g.Log().Infof(ctx, "%s 连接已结束", srcName)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "EOF"),
					time.Now().Add(time.Second),
				)
				finalErr = &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "EOF"}
			} else if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
				g.Log().Infof(ctx, "%s 连接已关闭", srcName)
				finalErr = &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "connection closed"}
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				g.Log().Infof(ctx, "%s 上下文结束: %v", srcName, err)
				finalErr = &websocket.CloseError{Code: websocket.CloseGoingAway, Text: err.Error()}
			} else if websocket.IsUnexpectedCloseError(err) {
				g.Log().Warningf(ctx, "%s 连接异常关闭: %v", srcName, err)
				_ = dst.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "unexpected close"),
					time.Now().Add(time.Second),
				)
			}
			return
		}

		// 检测客户端 ACK 消息
		if srcName == "client" && msgType == websocket.BinaryMessage {
			if frame, err := parseSAUCFrame(msg); err == nil {
				if frame.header.MessageType == saucMsgTypeClientAck {
					g.Log().Infof(ctx, "收到客户端ACK确认，准备关闭连接")
					// 收到客户端确认，可以安全关闭
					return
				}
				// 检查是否是带 FINAL_PACKET 标志的消息
				if (frame.header.Flags & messageFlagFinalPacket) == messageFlagFinalPacket {
					g.Log().Infof(ctx, "%s 收到最终音频包，准备结束录音", srcName)
					waitingForTaskComplete = true
					recorderActive = false // 停止录音
				}
			}
		}

		if recorderActive && msgType == websocket.BinaryMessage {
			pcm, handled, err := extractPCMFromFrame(msg)
			if err != nil {
				g.Log().Warningf(ctx, "%s 录音帧解析失败: %v", srcName, err)
				recorderActive = false
			} else if len(pcm) > 0 {
				if err := recorder.Append(pcm); err != nil {
					g.Log().Warningf(ctx, "%s 录音写入失败: %v", srcName, err)
					recorderActive = false
				}
			} else if handled {
				// 帧已解析但无需写入（例如 full client request），直接跳过。
			} else {
				g.Log().Debugf(ctx, "%s 收到非音频二进制帧，message_type 未处理", srcName)
			}
		}

		if err := dst.WriteMessage(msgType, msg); err != nil {
			finalErr = err
			return
		}

		// 如果这是服务器发来的带有 FINAL_PACKET 标志的消息
		// 说明服务器已经发送了所有转写结果，可以开始处理录音文件了
		if srcName == "upstream" && msgType == websocket.BinaryMessage && serverFinalReceivedCh != nil {
			if frame, err := parseSAUCFrame(msg); err == nil {
				if frame.header.MessageType == saucMsgTypeFullServerResponse && (frame.header.Flags&messageFlagFinalPacket) == messageFlagFinalPacket {
					g.Log().Infof(ctx, "服务器已发送最终转写结果，触发录音处理信号")
					select {
					case serverFinalReceivedCh <- struct{}{}:
					default:
						// 通道已满或已关闭，忽略
					}
				}
			}
		}
	}
}

func sendTaskCompleteMessage(ctx context.Context, conn *websocket.Conn, result *RecordingResult) error {
	// 构造任务信息
	taskInfo := g.Map{
		"taskId":    result.ConnectID,
		"connectId": result.ConnectID,
		"filePath":  result.FilePath,
		"fileSize":  result.Size,
		"duration":  result.EndedAt.Sub(result.StartedAt).Seconds(),
		"startedAt": result.StartedAt.Format(time.RFC3339),
		"endedAt":   result.EndedAt.Format(time.RFC3339),
	}

	// 序列化任务信息为 JSON
	payload, err := json.Marshal(taskInfo)
	if err != nil {
		return err
	}

	message := buildSAUCMessage(saucMsgTypeTaskComplete, saucSerializationJSON, saucCompressionNone, 0, payload)
	return conn.WriteMessage(websocket.BinaryMessage, message)
}
