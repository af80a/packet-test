package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Mode flags
	serverMode := flag.Bool("server", false, "Run in server mode")
	clientMode := flag.Bool("client", false, "Run in client mode")

	// Connection flags
	host := flag.String("host", "localhost", "Server address (client mode)")
	port := flag.Int("port", 9999, "UDP port")

	// Client flags
	packetSize := flag.Int("packet-size", 128, "Packet payload size in bytes")
	rate := flag.Int("rate", 64, "Packets per second")
	duration := flag.Int("duration", 30, "Test duration in seconds")
	output := flag.String("output", "", "CSV filename (auto-generated if empty)")
	burst := flag.Bool("burst", false, "Send packets in bursts (exposes WiFi buffering)")
	burstSize := flag.Int("burst-size", 10, "Packets per burst (with --burst)")
	noPlot := flag.Bool("no-plot", false, "Don't generate HTML plot or open browser")
	lateThreshold := flag.Float64("late-threshold", 100, "Packets above this latency (ms) are counted as late")

	// Plot flag
	plotFile := flag.String("plot", "", "Generate HTML chart from CSV file")

	flag.Parse()

	// Plot mode
	if *plotFile != "" {
		if err := GeneratePlot(*plotFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validate mode selection
	if *serverMode && *clientMode {
		fmt.Fprintln(os.Stderr, "Error: cannot use both --server and --client")
		os.Exit(1)
	}

	if !*serverMode && !*clientMode {
		fmt.Fprintln(os.Stderr, "Error: must specify --server or --client mode")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate packet size
	if *packetSize < HeaderSize {
		fmt.Fprintf(os.Stderr, "Error: packet-size must be at least %d bytes\n", HeaderSize)
		os.Exit(1)
	}

	// Run selected mode
	var err error
	if *serverMode {
		err = RunServer(*port)
	} else {
		cfg := ClientConfig{
			Host:          *host,
			Port:          *port,
			PacketSize:    *packetSize,
			Rate:          *rate,
			Duration:      *duration,
			OutputFile:    *output,
			Burst:         *burst,
			BurstSize:     *burstSize,
			NoPlot:        *noPlot,
			LateThreshold: *lateThreshold,
		}
		err = RunClient(cfg)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
