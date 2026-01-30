package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// RunServer starts the UDP echo server
func RunServer(port int) error {
	addr := fmt.Sprintf(":%d", port)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer conn.Close()

	fmt.Printf("UDP server listening on port %d\n", port)
	fmt.Println("Press Ctrl+C to stop")

	buf := make([]byte, 65535)
	clients := make(map[string]bool)

	for {
		n, clientAddr, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			continue
		}
		recvTime := time.Now()

		// Log new clients
		addrStr := clientAddr.String()
		if !clients[addrStr] {
			clients[addrStr] = true
			fmt.Printf("New client connected: %s\n", addrStr)
		}

		// Stamp server processing time into the response (if packet is large enough)
		if n >= HeaderSize {
			procNs := time.Since(recvTime).Nanoseconds()
			binary.BigEndian.PutUint64(buf[16:24], uint64(procNs))
		}

		// Echo the packet back immediately
		_, err = conn.WriteTo(buf[:n], clientAddr)
		if err != nil {
			fmt.Printf("Write error to %s: %v\n", addrStr, err)
		}
	}
}
