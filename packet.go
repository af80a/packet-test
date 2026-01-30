package main

import (
	"encoding/binary"
	"time"
)

const (
	SeqNumSize    = 8
	TimestampSize = 8
	HeaderSize    = SeqNumSize + TimestampSize
)

// Packet represents a UDP test packet
type Packet struct {
	SeqNum    uint64
	Timestamp int64 // Unix nanoseconds
	Payload   []byte
}

// Encode serializes the packet into bytes
func (p *Packet) Encode(size int) []byte {
	buf := make([]byte, size)
	binary.BigEndian.PutUint64(buf[0:8], p.SeqNum)
	binary.BigEndian.PutUint64(buf[8:16], uint64(p.Timestamp))
	// Rest is padding (zeros)
	return buf
}

// DecodePacket deserializes bytes into a Packet
func DecodePacket(data []byte) *Packet {
	if len(data) < HeaderSize {
		return nil
	}
	return &Packet{
		SeqNum:    binary.BigEndian.Uint64(data[0:8]),
		Timestamp: int64(binary.BigEndian.Uint64(data[8:16])),
		Payload:   data[HeaderSize:],
	}
}

// NewPacket creates a new packet with the current timestamp
func NewPacket(seqNum uint64, size int) *Packet {
	return &Packet{
		SeqNum:    seqNum,
		Timestamp: time.Now().UnixNano(),
		Payload:   make([]byte, size-HeaderSize),
	}
}
