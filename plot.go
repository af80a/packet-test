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
            max-width: 1200px;
            margin: 0 auto;
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
    </div>

    <div class="chart-container">
        <canvas id="latencyChart"></canvas>
    </div>

    <div class="chart-container">
        <canvas id="lossChart"></canvas>
    </div>

    <script>
        const data = {{DATA_JSON}};

        // Latency chart
        new Chart(document.getElementById('latencyChart'), {
            type: 'line',
            data: {
                labels: data.map(d => d.seq),
                datasets: [{
                    label: 'Latency (ms)',
                    data: data.map(d => d.lost ? null : d.latency),
                    borderColor: '#00d9ff',
                    backgroundColor: 'rgba(0, 217, 255, 0.1)',
                    fill: true,
                    tension: 0.1,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    title: { display: true, text: 'Latency Over Time', color: '#eee' }
                },
                scales: {
                    x: {
                        title: { display: true, text: 'Packet Sequence', color: '#888' },
                        ticks: { color: '#888' },
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

        // Calculate packet loss over windows
        const windowSize = Math.max(10, Math.floor(data.length / 100));
        const lossData = [];
        for (let i = 0; i < data.length; i += windowSize) {
            const window = data.slice(i, i + windowSize);
            const lost = window.filter(d => d.lost).length;
            lossData.push({
                seq: i + windowSize/2,
                loss: (lost / window.length) * 100
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

	for i, record := range records[1:] { // Skip header
		if len(record) < 5 {
			continue
		}

		seq := record[0]
		latency, _ := strconv.ParseFloat(record[3], 64)
		lost := record[4] == "true"

		totalPackets++
		if lost {
			lostPackets++
		} else {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		}

		if i > 0 {
			dataJSON.WriteString(",")
		}
		dataJSON.WriteString(fmt.Sprintf(`{"seq":%s,"latency":%.2f,"lost":%t}`, seq, latency, lost))
	}
	dataJSON.WriteString("]")

	// Calculate summary stats
	lossPercent := float64(0)
	avgLatency := float64(0)
	if totalPackets > 0 {
		lossPercent = float64(lostPackets) / float64(totalPackets) * 100
	}
	if totalPackets-lostPackets > 0 {
		avgLatency = totalLatency / float64(totalPackets-lostPackets)
	}

	// Generate HTML
	html := htmlTemplate
	html = strings.Replace(html, "{{TOTAL_PACKETS}}", strconv.Itoa(totalPackets), 1)
	html = strings.Replace(html, "{{LOSS_PERCENT}}", fmt.Sprintf("%.2f", lossPercent), 1)
	html = strings.Replace(html, "{{AVG_LATENCY}}", fmt.Sprintf("%.1f", avgLatency), 1)
	html = strings.Replace(html, "{{MAX_LATENCY}}", fmt.Sprintf("%.1f", maxLatency), 1)
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
