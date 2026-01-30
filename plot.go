package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Packet Loss Test Results</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #1a1a2e;
            color: #eee;
        }
        h1 { color: #00d9ff; }
        .chart-container {
            background: #16213e;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        .stat-box {
            background: #16213e;
            padding: 15px;
            border-radius: 8px;
            text-align: center;
        }
        .stat-value {
            font-size: 2em;
            color: #00d9ff;
            font-weight: bold;
        }
        .stat-label {
            color: #888;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <h1>UDP Packet Loss Test Results</h1>

    <div class="stats">
        <div class="stat-box">
            <div class="stat-value">{{TOTAL_PACKETS}}</div>
            <div class="stat-label">Packets Sent</div>
        </div>
        <div class="stat-box">
            <div class="stat-value">{{LOSS_PERCENT}}%</div>
            <div class="stat-label">Packet Loss</div>
        </div>
        <div class="stat-box">
            <div class="stat-value">{{AVG_LATENCY}}ms</div>
            <div class="stat-label">Avg Latency</div>
        </div>
        <div class="stat-box">
            <div class="stat-value">{{MAX_LATENCY}}ms</div>
            <div class="stat-label">Max Latency</div>
        </div>
        <div class="stat-box">
            <div class="stat-value">{{AVG_NET_LATENCY}}</div>
            <div class="stat-label">Avg Net+Client</div>
        </div>
        <div class="stat-box">
            <div class="stat-value">{{AVG_SERVER_PROC}}</div>
            <div class="stat-label">Avg Server Proc</div>
        </div>
    </div>

    <div class="chart-container">
        <canvas id="latencyChart"></canvas>
    </div>

    <div class="chart-container">
        <canvas id="netLatencyChart"></canvas>
    </div>

    <div class="chart-container">
        <canvas id="serverProcChart"></canvas>
    </div>

    <div class="chart-container">
        <canvas id="throughputChart"></canvas>
    </div>

    <div class="chart-container">
        <canvas id="lossChart"></canvas>
    </div>

    <script>
        const data = {{DATA_JSON}};

        // Sort data by sequence number for proper display
        data.sort((a, b) => a.seq - b.seq);

        // Latency bar chart
        new Chart(document.getElementById('latencyChart'), {
            type: 'bar',
            data: {
                labels: data.map(d => d.seq),
                datasets: [{
                    label: 'Latency (ms)',
                    data: data.map(d => d.lost ? null : d.latency),
                    backgroundColor: data.map(d => {
                        if (d.lost) return '#ff6b6b';
                        if (d.latency > 50) return '#feca57';
                        return '#00d9ff';
                    }),
                    borderWidth: 0
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    title: { display: true, text: 'Latency Per Packet', color: '#eee' }
                },
                scales: {
                    x: {
                        title: { display: true, text: 'Packet Sequence', color: '#888' },
                        ticks: { color: '#888', maxTicksLimit: 20 },
                        grid: { color: '#333' }
                    },
                    y: {
                        title: { display: true, text: 'Latency (ms)', color: '#888' },
                        ticks: { color: '#888' },
                        grid: { color: '#333' }
                    }
                }
            }
        });

        const hasNet = data.some(d => d.net !== null);
        if (hasNet) {
            new Chart(document.getElementById('netLatencyChart'), {
                type: 'line',
                data: {
                    labels: data.map(d => d.seq),
                    datasets: [{
                        label: 'Net+Client (ms)',
                        data: data.map(d => d.lost ? null : d.net),
                        borderColor: '#4ecdc4',
                        backgroundColor: 'rgba(78, 205, 196, 0.2)',
                        pointRadius: 0,
                        spanGaps: true,
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        title: { display: true, text: 'Net+Client Latency Per Packet', color: '#eee' }
                    },
                    scales: {
                        x: {
                            title: { display: true, text: 'Packet Sequence', color: '#888' },
                            ticks: { color: '#888', maxTicksLimit: 20 },
                            grid: { color: '#333' }
                        },
                        y: {
                            title: { display: true, text: 'Latency (ms)', color: '#888' },
                            ticks: { color: '#888' },
                            grid: { color: '#333' }
                        }
                    }
                }
            });
        } else {
            document.getElementById('netLatencyChart').parentElement.style.display = 'none';
        }

        const hasServer = data.some(d => d.server !== null);
        if (hasServer) {
            new Chart(document.getElementById('serverProcChart'), {
                type: 'line',
                data: {
                    labels: data.map(d => d.seq),
                    datasets: [{
                        label: 'Server Proc (ms)',
                        data: data.map(d => d.lost ? null : d.server),
                        borderColor: '#feca57',
                        backgroundColor: 'rgba(254, 202, 87, 0.2)',
                        pointRadius: 0,
                        spanGaps: true,
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        title: { display: true, text: 'Server Processing Time Per Packet', color: '#eee' }
                    },
                    scales: {
                        x: {
                            title: { display: true, text: 'Packet Sequence', color: '#888' },
                            ticks: { color: '#888', maxTicksLimit: 20 },
                            grid: { color: '#333' }
                        },
                        y: {
                            title: { display: true, text: 'Time (ms)', color: '#888' },
                            ticks: { color: '#888' },
                            grid: { color: '#333' }
                        }
                    }
                }
            });
        } else {
            document.getElementById('serverProcChart').parentElement.style.display = 'none';
        }

        // Calculate throughput over time (packets per 500ms window)
        const receivedPackets = data.filter(d => !d.lost && d.recvTime > 0);
        if (receivedPackets.length > 0) {
            const minTime = Math.min(...receivedPackets.map(d => d.recvTime));
            const maxTime = Math.max(...receivedPackets.map(d => d.recvTime));
            const windowMs = 500; // 500ms windows
            const throughputData = [];

            for (let t = minTime; t < maxTime; t += windowMs) {
                const windowEnd = t + windowMs;
                const packetsInWindow = receivedPackets.filter(d => d.recvTime >= t && d.recvTime < windowEnd).length;
                const pps = (packetsInWindow / windowMs) * 1000; // packets per second
                throughputData.push({
                    time: ((t - minTime) / 1000).toFixed(1),
                    pps: pps
                });
            }

            // Throughput chart
            new Chart(document.getElementById('throughputChart'), {
                type: 'bar',
                data: {
                    labels: throughputData.map(d => d.time + 's'),
                    datasets: [{
                        label: 'Packets/sec',
                        data: throughputData.map(d => d.pps),
                        backgroundColor: throughputData.map(d => {
                            if (d.pps < 30) return '#ff6b6b';
                            if (d.pps < 50) return '#feca57';
                            return '#4ecdc4';
                        }),
                        borderWidth: 0
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        title: { display: true, text: 'Throughput Over Time (packets received per second)', color: '#eee' }
                    },
                    scales: {
                        x: {
                            title: { display: true, text: 'Time', color: '#888' },
                            ticks: { color: '#888' },
                            grid: { color: '#333' }
                        },
                        y: {
                            title: { display: true, text: 'Packets/sec', color: '#888' },
                            ticks: { color: '#888' },
                            grid: { color: '#333' },
                            min: 0
                        }
                    }
                }
            });
        }

        // Calculate packet loss over windows
        const windowSize = Math.max(10, Math.floor(data.length / 100));
        const lossData = [];
        for (let i = 0; i < data.length; i += windowSize) {
            const win = data.slice(i, i + windowSize);
            const lost = win.filter(d => d.lost).length;
            lossData.push({
                seq: i + windowSize/2,
                loss: (lost / win.length) * 100
            });
        }

        // Loss chart
        new Chart(document.getElementById('lossChart'), {
            type: 'bar',
            data: {
                labels: lossData.map(d => Math.floor(d.seq)),
                datasets: [{
                    label: 'Packet Loss %',
                    data: lossData.map(d => d.loss),
                    backgroundColor: lossData.map(d => d.loss > 1 ? '#ff6b6b' : '#4ecdc4'),
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    title: { display: true, text: 'Packet Loss Over Time', color: '#eee' }
                },
                scales: {
                    x: {
                        title: { display: true, text: 'Packet Sequence', color: '#888' },
                        ticks: { color: '#888' },
                        grid: { color: '#333' }
                    },
                    y: {
                        title: { display: true, text: 'Loss %', color: '#888' },
                        ticks: { color: '#888' },
                        grid: { color: '#333' },
                        min: 0
                    }
                }
            }
        });
    </script>
</body>
</html>`

// GeneratePlot reads a CSV file and generates an HTML chart
func GeneratePlot(csvFile string) error {
	// Read CSV
	file, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("failed to open CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data rows")
	}

	// Parse data and calculate stats
	var dataJSON strings.Builder
	dataJSON.WriteString("[")

	var totalPackets, lostPackets int
	var totalLatency, maxLatency float64
	var totalNet, totalServer float64
	var receivedCount int

	header := records[0]
	colIndex := make(map[string]int, len(header))
	for i, col := range header {
		colIndex[strings.TrimSpace(col)] = i
	}

	seqIdx, ok := colIndex["seq"]
	if !ok {
		return fmt.Errorf("CSV missing required column: seq")
	}
	recvIdx, ok := colIndex["recv_time"]
	if !ok {
		return fmt.Errorf("CSV missing required column: recv_time")
	}
	latIdx, ok := colIndex["latency_ms"]
	if !ok {
		return fmt.Errorf("CSV missing required column: latency_ms")
	}
	lostIdx, ok := colIndex["lost"]
	if !ok {
		return fmt.Errorf("CSV missing required column: lost")
	}
	netIdx, hasNet := colIndex["net_latency_ms"]
	serverIdx, hasServer := colIndex["server_proc_ms"]

	for i, record := range records[1:] { // Skip header
		if seqIdx >= len(record) || recvIdx >= len(record) || latIdx >= len(record) || lostIdx >= len(record) {
			continue
		}

		seq := record[seqIdx]
		recvTime := record[recvIdx]
		latency, _ := strconv.ParseFloat(record[latIdx], 64)
		lost := record[lostIdx] == "true"

		netJSON := "null"
		serverJSON := "null"
		if hasNet && netIdx < len(record) {
			if netVal, err := strconv.ParseFloat(record[netIdx], 64); err == nil {
				netJSON = fmt.Sprintf("%.2f", netVal)
				if !lost {
					totalNet += netVal
				}
			}
		}
		if hasServer && serverIdx < len(record) {
			if serverVal, err := strconv.ParseFloat(record[serverIdx], 64); err == nil {
				serverJSON = fmt.Sprintf("%.2f", serverVal)
				if !lost {
					totalServer += serverVal
				}
			}
		}

		totalPackets++
		if lost {
			lostPackets++
		} else {
			receivedCount++
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		}

		if i > 0 {
			dataJSON.WriteString(",")
		}
		dataJSON.WriteString(fmt.Sprintf(`{"seq":%s,"recvTime":%s,"latency":%.2f,"net":%s,"server":%s,"lost":%t}`,
			seq, recvTime, latency, netJSON, serverJSON, lost))
	}
	dataJSON.WriteString("]")

	// Calculate summary stats
	lossPercent := float64(0)
	avgLatency := float64(0)
	avgNet := "N/A"
	avgServer := "N/A"
	if totalPackets > 0 {
		lossPercent = float64(lostPackets) / float64(totalPackets) * 100
	}
	if receivedCount > 0 {
		avgLatency = totalLatency / float64(receivedCount)
		if hasNet {
			avgNet = fmt.Sprintf("%.1fms", totalNet/float64(receivedCount))
		}
		if hasServer {
			avgServer = fmt.Sprintf("%.1fms", totalServer/float64(receivedCount))
		}
	}

	// Generate HTML
	html := htmlTemplate
	html = strings.Replace(html, "{{TOTAL_PACKETS}}", strconv.Itoa(totalPackets), 1)
	html = strings.Replace(html, "{{LOSS_PERCENT}}", fmt.Sprintf("%.2f", lossPercent), 1)
	html = strings.Replace(html, "{{AVG_LATENCY}}", fmt.Sprintf("%.1f", avgLatency), 1)
	html = strings.Replace(html, "{{MAX_LATENCY}}", fmt.Sprintf("%.1f", maxLatency), 1)
	html = strings.Replace(html, "{{AVG_NET_LATENCY}}", avgNet, 1)
	html = strings.Replace(html, "{{AVG_SERVER_PROC}}", avgServer, 1)
	html = strings.Replace(html, "{{DATA_JSON}}", dataJSON.String(), 1)

	// Write output file
	outputFile := strings.TrimSuffix(csvFile, ".csv") + ".html"
	err = os.WriteFile(outputFile, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	fmt.Printf("Generated %s\n", outputFile)
	return nil
}
