package meetingRecord

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
)

type saucFrame struct {
	header  saucHeader
	payload []byte
}

type saucHeader struct {
	Version       uint8
	HeaderSize    uint8
	MessageType   uint8
	Flags         uint8
	Serialization uint8
	Compression   uint8
	Reserved      uint8
}

const (
	saucMsgTypeFullClientRequest      = 0x1
	saucMsgTypeAudioOnlyClientRequest = 0x2
	saucMsgTypeFullServerResponse     = 0x9
	saucMsgTypeTaskComplete           = 0xa
	saucMsgTypeClientAck              = 0xb

	saucSerializationNone = 0x0
	saucSerializationJSON = 0x1

	saucCompressionNone = 0x0
	saucCompressionGzip = 0x1

	messageFlagFinalPacket = 0x2 // FINAL_PACKET 标志位
)

func parseSAUCFrame(data []byte) (*saucFrame, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("frame too short: %d", len(data))
	}
	hdr := saucHeader{
		Version:       data[0] >> 4,
		HeaderSize:    data[0] & 0x0F,
		MessageType:   data[1] >> 4,
		Flags:         data[1] & 0x0F,
		Serialization: data[2] >> 4,
		Compression:   data[2] & 0x0F,
		Reserved:      data[3],
	}
	payloadSize := int(binary.BigEndian.Uint32(data[4:8]))
	if len(data) < 8+payloadSize {
		return nil, fmt.Errorf("frame payload truncated: want %d bytes, have %d", payloadSize, len(data)-8)
	}
	payload := data[8 : 8+payloadSize]
	return &saucFrame{header: hdr, payload: payload}, nil
}

func (f *saucFrame) payloadBytes() ([]byte, error) {
	switch f.header.Compression {
	case saucCompressionNone:
		return f.payload, nil
	case saucCompressionGzip:
		reader, err := gzip.NewReader(bytes.NewReader(f.payload))
		if err != nil {
			return nil, fmt.Errorf("gzip reader init failed: %w", err)
		}
		defer reader.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, reader); err != nil {
			return nil, fmt.Errorf("gzip decompress failed: %w", err)
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported compression: %d", f.header.Compression)
	}
}

// extractPCMFromFrame parses the SAUC frame and returns raw PCM bytes when the
// frame carries audio data. The boolean return indicates whether the frame was
// handled (true) or ignored (false).
func extractPCMFromFrame(data []byte) ([]byte, bool, error) {
	frame, err := parseSAUCFrame(data)
	if err != nil {
		return nil, false, err
	}
	switch frame.header.MessageType {
	case saucMsgTypeAudioOnlyClientRequest:
		if frame.header.Serialization != saucSerializationNone {
			return nil, true, fmt.Errorf("unsupported audio serialization: %d", frame.header.Serialization)
		}
		payload, err := frame.payloadBytes()
		if err != nil {
			return nil, true, err
		}
		return payload, true, nil
	case saucMsgTypeFullClientRequest:
		// Full client request carries metadata. We skip recording but mark as handled.
		return nil, true, nil
	default:
		return nil, false, nil
	}
}

// buildSAUCMessage 构造 SAUC 协议消息
func buildSAUCMessage(messageType, serialization, compression, flags uint8, payload []byte) []byte {
	headerSize := uint8(0x1) // 4 字节
	payloadSize := uint32(len(payload))

	buffer := make([]byte, 8+len(payload))

	// 构造头部
	buffer[0] = (0x1 << 4) | headerSize // Protocol version 0x1
	buffer[1] = (messageType << 4) | (flags & 0x0f)
	buffer[2] = (serialization << 4) | (compression & 0x0f)
	buffer[3] = 0x00 // Reserved

	// 写入 payload 大小（大端序）
	binary.BigEndian.PutUint32(buffer[4:8], payloadSize)

	// 复制 payload
	copy(buffer[8:], payload)

	return buffer
}
