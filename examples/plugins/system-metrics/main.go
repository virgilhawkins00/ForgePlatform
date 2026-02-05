// Example Forge plugin: System Metrics Collector
//
// This plugin demonstrates how to use the Forge SDK to:
// - Collect system metrics (CPU, memory, disk)
// - Log messages to the Forge runtime
// - Use configuration values
// - Handle periodic tick events
//
// Build with TinyGo:
//   tinygo build -o system-metrics.wasm -target=wasi -scheduler=none .
//
// Install:
//   forge plugin install ./system-metrics.wasm

package main

import (
	"github.com/forge-platform/forge/pkg/sdk"
)

// SystemMetricsPlugin collects system metrics.
type SystemMetricsPlugin struct {
	interval int // collection interval in seconds
}

// Ensure we implement the required interfaces.
var (
	_ sdk.Plugin          = (*SystemMetricsPlugin)(nil)
	_ sdk.TickHandler     = (*SystemMetricsPlugin)(nil)
	_ sdk.MetricCollector = (*SystemMetricsPlugin)(nil)
	_ sdk.ConfigProvider  = (*SystemMetricsPlugin)(nil)
)

func (p *SystemMetricsPlugin) Name() string {
	return "system-metrics"
}

func (p *SystemMetricsPlugin) Version() string {
	return "1.0.0"
}

func (p *SystemMetricsPlugin) Init() error {
	sdk.Info("System metrics plugin initialized")

	// Load configuration
	if interval, ok := sdk.GetConfig("interval"); ok {
		sdk.Debug("Using configured interval: " + interval)
	} else {
		p.interval = 10 // default: 10 seconds
	}

	return nil
}

func (p *SystemMetricsPlugin) Cleanup() error {
	sdk.Info("System metrics plugin cleanup")
	return nil
}

// OnTick is called periodically by the Forge runtime.
func (p *SystemMetricsPlugin) OnTick() error {
	return p.CollectMetrics()
}

// CollectMetrics collects system metrics.
func (p *SystemMetricsPlugin) CollectMetrics() error {
	// In a real plugin, we would use WASI to read /proc files
	// or call OS-specific APIs. For this example, we use mock data.

	// CPU usage (mock)
	cpuUsage := 45.2
	sdk.RecordMetric("cpu.usage", cpuUsage)
	sdk.RecordMetricWithTags("cpu.usage", cpuUsage, map[string]string{
		"host": "localhost",
		"core": "all",
	})

	// Memory usage (mock)
	memUsage := 62.5
	sdk.RecordMetric("memory.usage", memUsage)

	// Disk usage (mock)
	diskUsage := 34.8
	sdk.RecordMetricWithTags("disk.usage", diskUsage, map[string]string{
		"mount": "/",
	})

	sdk.Debug("Collected system metrics")
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration.
func (p *SystemMetricsPlugin) ConfigSchema() string {
	return `{
  "type": "object",
  "properties": {
    "interval": {
      "type": "integer",
      "description": "Collection interval in seconds",
      "default": 10,
      "minimum": 1
    },
    "collect_cpu": {
      "type": "boolean",
      "description": "Collect CPU metrics",
      "default": true
    },
    "collect_memory": {
      "type": "boolean",
      "description": "Collect memory metrics",
      "default": true
    },
    "collect_disk": {
      "type": "boolean",
      "description": "Collect disk metrics",
      "default": true
    }
  }
}`
}

// Configure applies the plugin configuration.
func (p *SystemMetricsPlugin) Configure(config []byte) error {
	sdk.Debug("Received configuration")
	// In a real plugin, parse JSON config here
	return nil
}

func main() {
	// Register the plugin with the Forge runtime
	sdk.Register(&SystemMetricsPlugin{
		interval: 10,
	})
}

