package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PacketRecord stores data for a single packet
type PacketRecord struct {
	SeqNum       uint64
	SentTime     int64 // Unix nanoseconds
	RecvTime     int64 // Unix nanoseconds, 0 if lost
	LatencyMs    float64
	ServerProcMs float64
	NetLatencyMs float64
	Lost         bool
	Late         bool
}

// Stats tracks packet statistics
type Stats struct {
	mu      sync.Mutex
	records map[uint64]*PacketRecord

	sent     uint64
	received uint64
	late     uint64

	lateThreshold float64 // milliseconds

	latencies    []float64
	netLatencies []float64
	serverProc   []float64
	minLat       float64
	maxLat       float64
	sumLat       float64
	minNet       float64
	maxNet       float64
	sumNet       float64
	minServer    float64
	maxServer    float64
	sumServer    float64

	windowStartNs    int64
	windowSent       uint64
	windowReceived   uint64
	windowLate       uint64
	windowLatencies  []float64
	windowNetLatency []float64
	windowServerProc []float64

	lastPrintTime time.Time
	startTime     time.Time
}

// NewStats creates a new Stats tracker
func NewStats(lateThreshold float64) *Stats {
	now := time.Now()
	return &Stats{
		records:       make(map[uint64]*PacketRecord),
		lateThreshold: lateThreshold,
		minLat:        math.MaxFloat64,
		minNet:        math.MaxFloat64,
		minServer:     math.MaxFloat64,
		maxLat:        0,
		lastPrintTime: now,
		startTime:     now,
		windowStartNs: now.UnixNano(),
	}
}

// RecordSent records a sent packet
func (s *Stats) RecordSent(seqNum uint64, sentTime int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent++
	s.windowSent++
	s.records[seqNum] = &PacketRecord{
		SeqNum:   seqNum,
		SentTime: sentTime,
		Lost:     true, // Assume lost until we receive response
	}
}

// RecordReceived records a received packet response
func (s *Stats) RecordReceived(seqNum uint64, recvTime int64, serverProcNs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[seqNum]
	if !exists {
		return
	}

	if record.Lost { // Only count first response
		record.RecvTime = recvTime
		record.LatencyMs = float64(recvTime-record.SentTime) / float64(time.Millisecond)
		record.ServerProcMs = float64(serverProcNs) / float64(time.Millisecond)
		netLatency := record.LatencyMs - record.ServerProcMs
		if netLatency < 0 {
			netLatency = 0
		}
		record.NetLatencyMs = netLatency
		record.Lost = false

		// Check if packet is late
		if record.LatencyMs > s.lateThreshold {
			record.Late = true
			s.late++
		}

		s.received++
		s.latencies = append(s.latencies, record.LatencyMs)
		s.sumLat += record.LatencyMs
		s.netLatencies = append(s.netLatencies, record.NetLatencyMs)
		s.sumNet += record.NetLatencyMs
		s.serverProc = append(s.serverProc, record.ServerProcMs)
		s.sumServer += record.ServerProcMs

		if record.LatencyMs < s.minLat {
			s.minLat = record.LatencyMs
		}
		if record.LatencyMs > s.maxLat {
			s.maxLat = record.LatencyMs
		}
		if record.NetLatencyMs < s.minNet {
			s.minNet = record.NetLatencyMs
		}
		if record.NetLatencyMs > s.maxNet {
			s.maxNet = record.NetLatencyMs
		}
		if record.ServerProcMs < s.minServer {
			s.minServer = record.ServerProcMs
		}
		if record.ServerProcMs > s.maxServer {
			s.maxServer = record.ServerProcMs
		}

		if record.SentTime >= s.windowStartNs {
			s.windowReceived++
			if record.Late {
				s.windowLate++
			}
			s.windowLatencies = append(s.windowLatencies, record.LatencyMs)
			s.windowNetLatency = append(s.windowNetLatency, record.NetLatencyMs)
			s.windowServerProc = append(s.windowServerProc, record.ServerProcMs)
		}
	}
}

// PrintInterval prints interval stats if 5 seconds have passed
func (s *Stats) PrintInterval() {
	if time.Since(s.lastPrintTime) < 5*time.Second {
		return
	}
	now := time.Now()
	s.mu.Lock()

	elapsed := time.Since(s.startTime)
	windowSent := s.windowSent
	windowReceived := s.windowReceived
	windowLate := s.windowLate
	windowLatencies := append([]float64(nil), s.windowLatencies...)
	windowNet := append([]float64(nil), s.windowNetLatency...)
	windowServer := append([]float64(nil), s.windowServerProc...)

	s.windowStartNs = now.UnixNano()
	s.windowSent = 0
	s.windowReceived = 0
	s.windowLate = 0
	s.windowLatencies = s.windowLatencies[:0]
	s.windowNetLatency = s.windowNetLatency[:0]
	s.windowServerProc = s.windowServerProc[:0]
	s.lastPrintTime = now
	s.mu.Unlock()

	secs := int(elapsed.Seconds())
	loss := float64(0)
	if windowSent > 0 {
		loss = float64(windowSent-windowReceived) / float64(windowSent) * 100
	}

	minLat, avgLat, maxLat, jitter := calcStats(windowLatencies)
	avgNet := avg(windowNet)
	avgServer := avg(windowServer)

	spike := ""
	if jitter > 10 {
		spike = "  << spike"
	}

	fmt.Printf("[%ds] Win Loss: %.1f%%  Late: %d  RTT: %.0f/%.0f/%.0fms  Jitter: %.0fms  Net: %.0fms  Srv: %.0fms%s\n",
		secs, loss, windowLate, minLat, avgLat, maxLat, jitter, avgNet, avgServer, spike)
}

// PrintSummary prints the final summary
func (s *Stats) PrintSummary() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("\n--- Summary ---")

	lost := s.sent - s.received
	lossPercent := float64(0)
	latePercent := float64(0)
	if s.sent > 0 {
		lossPercent = float64(lost) / float64(s.sent) * 100
		latePercent = float64(s.late) / float64(s.sent) * 100
	}

	fmt.Printf("Packets: %d sent, %d received, %d lost (%.2f%%), %d late (%.2f%%)\n",
		s.sent, s.received, lost, lossPercent, s.late, latePercent)
	fmt.Printf("Late threshold: %.0fms\n", s.lateThreshold)

	if len(s.latencies) > 0 {
		avgLat := s.sumLat / float64(len(s.latencies))

		var jitterSum float64
		for _, lat := range s.latencies {
			jitterSum += math.Abs(lat - avgLat)
		}
		jitter := jitterSum / float64(len(s.latencies))

		p50, p90, p99 := percentiles(s.latencies, 50, 90, 99)
		fmt.Printf("RTT: min=%.0fms avg=%.0fms max=%.0fms p50=%.0fms p90=%.0fms p99=%.0fms\n",
			s.minLat, avgLat, s.maxLat, p50, p90, p99)
		fmt.Printf("Jitter: %.0fms average\n", jitter)
	} else {
		fmt.Println("RTT: no data (all packets lost)")
	}

	if len(s.netLatencies) > 0 {
		avgNet := s.sumNet / float64(len(s.netLatencies))
		p50, p90, p99 := percentiles(s.netLatencies, 50, 90, 99)
		fmt.Printf("Net+Client: min=%.0fms avg=%.0fms max=%.0fms p50=%.0fms p90=%.0fms p99=%.0fms\n",
			s.minNet, avgNet, s.maxNet, p50, p90, p99)
	}

	if len(s.serverProc) > 0 {
		avgServer := s.sumServer / float64(len(s.serverProc))
		fmt.Printf("Server proc: min=%.0fms avg=%.0fms max=%.0fms\n",
			s.minServer, avgServer, s.maxServer)
	} else {
		fmt.Println("Server proc: no data")
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

func calcStats(values []float64) (min, avg, max, jitter float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}

	min = math.MaxFloat64
	max = 0
	var sum float64
	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = sum / float64(len(values))

	var jitterSum float64
	for _, v := range values {
		jitterSum += math.Abs(v - avg)
	}
	jitter = jitterSum / float64(len(values))

	return
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentiles(values []float64, p50, p90, p99 float64) (float64, float64, float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	return percentile(sorted, p50), percentile(sorted, p90), percentile(sorted, p99)
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	pos := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
