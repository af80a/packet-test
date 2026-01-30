package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// PacketRecord stores data for a single packet
type PacketRecord struct {
	SeqNum    uint64
	SentTime  int64 // Unix nanoseconds
	RecvTime  int64 // Unix nanoseconds, 0 if lost
	LatencyMs float64
	Lost      bool
}

// Stats tracks packet statistics
type Stats struct {
	mu      sync.Mutex
	records map[uint64]*PacketRecord

	sent     uint64
	received uint64

	latencies []float64
	minLat    float64
	maxLat    float64
	sumLat    float64

	lastPrintTime time.Time
	startTime     time.Time
}

// NewStats creates a new Stats tracker
func NewStats() *Stats {
	return &Stats{
		records:       make(map[uint64]*PacketRecord),
		minLat:        math.MaxFloat64,
		maxLat:        0,
		lastPrintTime: time.Now(),
		startTime:     time.Now(),
	}
}

// RecordSent records a sent packet
func (s *Stats) RecordSent(seqNum uint64, sentTime int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent++
	s.records[seqNum] = &PacketRecord{
		SeqNum:   seqNum,
		SentTime: sentTime,
		Lost:     true, // Assume lost until we receive response
	}
}

// RecordReceived records a received packet response
func (s *Stats) RecordReceived(seqNum uint64, recvTime int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[seqNum]
	if !exists {
		return
	}

	if record.Lost { // Only count first response
		record.RecvTime = recvTime
		record.LatencyMs = float64(recvTime-record.SentTime) / float64(time.Millisecond)
		record.Lost = false

		s.received++
		s.latencies = append(s.latencies, record.LatencyMs)
		s.sumLat += record.LatencyMs

		if record.LatencyMs < s.minLat {
			s.minLat = record.LatencyMs
		}
		if record.LatencyMs > s.maxLat {
			s.maxLat = record.LatencyMs
		}
	}
}

// GetIntervalStats returns stats for printing at intervals
func (s *Stats) GetIntervalStats() (elapsed time.Duration, loss float64, minLat, avgLat, maxLat, jitter float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed = time.Since(s.startTime)

	if s.sent == 0 {
		return elapsed, 0, 0, 0, 0, 0
	}

	lost := s.sent - s.received
	loss = float64(lost) / float64(s.sent) * 100

	if len(s.latencies) == 0 {
		return elapsed, loss, 0, 0, 0, 0
	}

	minLat = s.minLat
	maxLat = s.maxLat
	avgLat = s.sumLat / float64(len(s.latencies))

	// Calculate jitter (average deviation from mean)
	var jitterSum float64
	for _, lat := range s.latencies {
		jitterSum += math.Abs(lat - avgLat)
	}
	jitter = jitterSum / float64(len(s.latencies))

	return
}

// PrintInterval prints interval stats if 5 seconds have passed
func (s *Stats) PrintInterval() {
	if time.Since(s.lastPrintTime) < 5*time.Second {
		return
	}
	s.lastPrintTime = time.Now()

	elapsed, loss, minLat, avgLat, maxLat, jitter := s.GetIntervalStats()
	secs := int(elapsed.Seconds())

	spike := ""
	if jitter > 10 {
		spike = "  << spike"
	}

	fmt.Printf("[%ds] Loss: %.1f%%  Latency: %.0f/%.0f/%.0fms  Jitter: %.0fms%s\n",
		secs, loss, minLat, avgLat, maxLat, jitter, spike)
}

// PrintSummary prints the final summary
func (s *Stats) PrintSummary() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("\n--- Summary ---")

	lost := s.sent - s.received
	lossPercent := float64(0)
	if s.sent > 0 {
		lossPercent = float64(lost) / float64(s.sent) * 100
	}

	fmt.Printf("Packets: %d sent, %d received, %d lost (%.2f%%)\n",
		s.sent, s.received, lost, lossPercent)

	if len(s.latencies) > 0 {
		avgLat := s.sumLat / float64(len(s.latencies))

		var jitterSum float64
		for _, lat := range s.latencies {
			jitterSum += math.Abs(lat - avgLat)
		}
		jitter := jitterSum / float64(len(s.latencies))

		fmt.Printf("Latency: min=%.0fms avg=%.0fms max=%.0fms\n",
			s.minLat, avgLat, s.maxLat)
		fmt.Printf("Jitter: %.0fms average\n", jitter)
	} else {
		fmt.Println("Latency: no data (all packets lost)")
	}
}

// GetRecords returns all packet records for CSV export
func (s *Stats) GetRecords() []*PacketRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]*PacketRecord, 0, len(s.records))
	for _, r := range s.records {
		records = append(records, r)
	}
	return records
}
