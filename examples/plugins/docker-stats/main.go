// Example Forge plugin: Docker Stats Collector
//
// This plugin demonstrates how to use the HTTP capability to
// collect metrics from the Docker API.
//
// Build with TinyGo:
//   tinygo build -o docker-stats.wasm -target=wasi -scheduler=none .
//
// Install:
//   forge plugin install ./docker-stats.wasm

package main

import (
	"github.com/forge-platform/forge/pkg/sdk"
)

// DockerStatsPlugin collects Docker container metrics.
type DockerStatsPlugin struct {
	dockerHost string
}

var (
	_ sdk.Plugin          = (*DockerStatsPlugin)(nil)
	_ sdk.TickHandler     = (*DockerStatsPlugin)(nil)
	_ sdk.MetricCollector = (*DockerStatsPlugin)(nil)
)

func (p *DockerStatsPlugin) Name() string {
	return "docker-stats"
}

func (p *DockerStatsPlugin) Version() string {
	return "1.0.0"
}

func (p *DockerStatsPlugin) Init() error {
	sdk.Info("Docker stats plugin initialized")

	// Get Docker host from config
	if host, ok := sdk.GetConfig("docker_host"); ok {
		p.dockerHost = host
	} else {
		p.dockerHost = "http://localhost:2375"
	}

	sdk.Debug("Docker host: " + p.dockerHost)
	return nil
}

func (p *DockerStatsPlugin) Cleanup() error {
	sdk.Info("Docker stats plugin cleanup")
	return nil
}

func (p *DockerStatsPlugin) OnTick() error {
	return p.CollectMetrics()
}

func (p *DockerStatsPlugin) CollectMetrics() error {
	// List containers
	resp, err := sdk.HTTPGet(p.dockerHost + "/containers/json")
	if err != nil {
		sdk.Error("Failed to list containers: " + err.Error())
		return err
	}

	if resp.StatusCode != 200 {
		sdk.Warn("Docker API returned non-200 status")
		return nil
	}

	// In a real plugin, we would parse the JSON response
	// and collect stats for each container
	sdk.Debug("Got container list from Docker API")

	// For this example, record mock metrics
	sdk.RecordMetricWithTags("docker.container.count", 5, map[string]string{
		"status": "running",
	})

	sdk.RecordMetricWithTags("docker.container.cpu", 12.5, map[string]string{
		"container": "nginx",
	})

	sdk.RecordMetricWithTags("docker.container.memory", 256.0, map[string]string{
		"container": "nginx",
		"unit":      "MB",
	})

	return nil
}

func main() {
	sdk.Register(&DockerStatsPlugin{})
}

