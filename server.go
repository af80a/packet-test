package main

import (
	"fmt"
	"net"
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

		// Log new clients
		addrStr := clientAddr.String()
		if !clients[addrStr] {
			clients[addrStr] = true
			fmt.Printf("New client connected: %s\n", addrStr)
		}

		// Echo the packet back immediately
		_, err = conn.WriteTo(buf[:n], clientAddr)
		if err != nil {
			fmt.Printf("Write error to %s: %v\n", addrStr, err)
		}
	}
}
