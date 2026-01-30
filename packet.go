package main

import (
	"encoding/binary"
)

const (
	SeqNumSize    = 8
	TimestampSize = 8
	ProcTimeSize  = 8
	HeaderSize    = SeqNumSize + TimestampSize + ProcTimeSize
)

// Packet represents a UDP test packet
type Packet struct {
	SeqNum        uint64
	Timestamp     int64 // Unix nanoseconds (client send time)
	ServerProcNs  int64 // Server processing duration in nanoseconds
	Payload       []byte
}

// Encode serializes the packet into bytes
func (p *Packet) Encode(size int) []byte {
	buf := make([]byte, size)
	binary.BigEndian.PutUint64(buf[0:8], p.SeqNum)
	binary.BigEndian.PutUint64(buf[8:16], uint64(p.Timestamp))
	binary.BigEndian.PutUint64(buf[16:24], uint64(p.ServerProcNs))
	// Rest is padding (zeros)
	return buf
}

// DecodePacket deserializes bytes into a Packet
func DecodePacket(data []byte) *Packet {
	if len(data) < HeaderSize {
		return nil
	}
	return &Packet{
		SeqNum:       binary.BigEndian.Uint64(data[0:8]),
		Timestamp:    int64(binary.BigEndian.Uint64(data[8:16])),
		ServerProcNs: int64(binary.BigEndian.Uint64(data[16:24])),
		Payload:      data[HeaderSize:],
	}
}

// NewPacket creates a new packet with the provided timestamp
func NewPacket(seqNum uint64, size int, timestamp int64) *Packet {
	return &Packet{
		SeqNum:       seqNum,
		Timestamp:    timestamp,
		ServerProcNs: 0,
		Payload:      make([]byte, size-HeaderSize),
	}
}
