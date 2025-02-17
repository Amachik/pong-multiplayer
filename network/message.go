package network

import (
	"bytes"
	"encoding/binary"
)

// MessageType defines our message types.
type MessageType uint8

const (
	MessageTypeHandshake        MessageType = 1
	MessageTypeHandshakeSuccess MessageType = 2
	MessageTypeError            MessageType = 3
	MessageTypeInputUpdate      MessageType = 4
	MessageTypeStateUpdate      MessageType = 5
	MessageTypePing             MessageType = 6
	MessageTypePong             MessageType = 7
)

// Message now includes a sequence number.
type Message struct {
	Type MessageType
	Seq  uint32
	Data []byte
}

// EncodeMessage produces a binary representation: 1 byte for type,
// 4 bytes for sequence, 2 bytes for data length, then the payload.
func EncodeMessage(msg Message) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, msg.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, msg.Seq); err != nil {
		return nil, err
	}
	length := uint16(len(msg.Data))
	if err := binary.Write(buf, binary.BigEndian, length); err != nil {
		return nil, err
	}
	if length > 0 {
		if _, err := buf.Write(msg.Data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// DecodeMessage converts a binary packet back into a Message.
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	buf := bytes.NewReader(data)
	if err := binary.Read(buf, binary.BigEndian, &msg.Type); err != nil {
		return msg, err
	}
	if err := binary.Read(buf, binary.BigEndian, &msg.Seq); err != nil {
		return msg, err
	}
	var length uint16
	if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
		return msg, err
	}
	msg.Data = make([]byte, length)
	if length > 0 {
		if n, err := buf.Read(msg.Data); err != nil || n != int(length) {
			return msg, err
		}
	}
	return msg, nil
}
