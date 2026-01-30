package main

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// ClientConfig holds client configuration
type ClientConfig struct {
	Host       string
	Port       int
	PacketSize int
	Rate       int
	Duration   int
	OutputFile string
	Burst      bool
	BurstSize  int
}

// RunClient runs the UDP test client
func RunClient(cfg ClientConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	if cfg.Burst {
		fmt.Printf("Sending %d pps in bursts of %d, %d byte packets to %s\n\n",
			cfg.Rate, cfg.BurstSize, cfg.PacketSize, addr)
	} else {
		fmt.Printf("Sending %d pps, %d byte packets to %s\n\n",
			cfg.Rate, cfg.PacketSize, addr)
	}

	stats := NewStats()

	// Start receiver goroutine
	done := make(chan struct{})
	go receivePackets(conn, stats, done)

	endTime := time.Now().Add(time.Duration(cfg.Duration) * time.Second)
	var seqNum uint64 = 1

	// Stats printing ticker
	statsTicker := time.NewTicker(100 * time.Millisecond)
	defer statsTicker.Stop()

	if cfg.Burst {
		// Burst mode: send BurstSize packets quickly, then pause
		// Calculate bursts per second to maintain overall rate
		burstsPerSecond := float64(cfg.Rate) / float64(cfg.BurstSize)
		burstInterval := time.Duration(float64(time.Second) / burstsPerSecond)
		burstTicker := time.NewTicker(burstInterval)
		defer burstTicker.Stop()

		for time.Now().Before(endTime) {
			select {
			case <-burstTicker.C:
				// Send burst of packets as fast as possible
				for i := 0; i < cfg.BurstSize; i++ {
					pkt := NewPacket(seqNum, cfg.PacketSize)
					data := pkt.Encode(cfg.PacketSize)
					stats.RecordSent(seqNum, pkt.Timestamp)
					conn.Write(data)
					seqNum++
				}

			case <-statsTicker.C:
				stats.PrintInterval()
			}
		}
	} else {
		// Steady mode: send packets at fixed interval
		interval := time.Second / time.Duration(cfg.Rate)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for time.Now().Before(endTime) {
			select {
			case <-ticker.C:
				pkt := NewPacket(seqNum, cfg.PacketSize)
				data := pkt.Encode(cfg.PacketSize)

				stats.RecordSent(seqNum, pkt.Timestamp)

				_, err := conn.Write(data)
				if err != nil {
					fmt.Printf("Send error: %v\n", err)
				}
				seqNum++

			case <-statsTicker.C:
				stats.PrintInterval()
			}
		}
	}

	// Wait a bit for final responses
	time.Sleep(500 * time.Millisecond)
	close(done)

	stats.PrintSummary()

	// Save to CSV if output file specified
	if cfg.OutputFile != "" {
		if err := saveCSV(cfg.OutputFile, stats); err != nil {
			return fmt.Errorf("failed to save CSV: %w", err)
		}
		fmt.Printf("\nResults saved to %s\n", cfg.OutputFile)
	}

	return nil
}

func receivePackets(conn net.Conn, stats *Stats, done chan struct{}) {
	buf := make([]byte, 65535)

	// Set read deadline to allow checking done channel
	for {
		select {
		case <-done:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				continue
			}

			recvTime := time.Now().UnixNano()
			pkt := DecodePacket(buf[:n])
			if pkt != nil {
				stats.RecordReceived(pkt.SeqNum, recvTime)
			}
		}
	}
}

func saveCSV(filename string, stats *Stats) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"seq", "sent_time", "recv_time", "latency_ms", "lost"})

	// Write records
	records := stats.GetRecords()
	for _, r := range records {
		writer.Write([]string{
			strconv.FormatUint(r.SeqNum, 10),
			strconv.FormatInt(r.SentTime/1000000, 10), // Convert to milliseconds
			strconv.FormatInt(r.RecvTime/1000000, 10),
			fmt.Sprintf("%.2f", r.LatencyMs),
			strconv.FormatBool(r.Lost),
		})
	}

	return nil
}
