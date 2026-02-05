// Example Forge plugin: Kubernetes Monitor
//
// This plugin demonstrates how to collect metrics from a Kubernetes cluster
// using the Kubernetes API. It collects node, pod, and deployment metrics.
//
// Build with TinyGo:
//   tinygo build -o kubernetes-monitor.wasm -target=wasi -scheduler=none .
//
// Install:
//   forge plugin install ./kubernetes-monitor.wasm
//
// Configuration:
//   kubernetes_api_url: https://kubernetes.default.svc
//   kubernetes_token: <service-account-token>
//   collect_nodes: true
//   collect_pods: true
//   collect_deployments: true

package main

import (
	"github.com/forge-platform/forge/pkg/sdk"
)

// KubernetesMonitorPlugin collects Kubernetes cluster metrics.
type KubernetesMonitorPlugin struct {
	apiURL     string
	token      string
	namespace  string
	collectAll bool
}

var (
	_ sdk.Plugin          = (*KubernetesMonitorPlugin)(nil)
	_ sdk.TickHandler     = (*KubernetesMonitorPlugin)(nil)
	_ sdk.MetricCollector = (*KubernetesMonitorPlugin)(nil)
)

func (p *KubernetesMonitorPlugin) Name() string {
	return "kubernetes-monitor"
}

func (p *KubernetesMonitorPlugin) Version() string {
	return "1.0.0"
}

func (p *KubernetesMonitorPlugin) Init() error {
	sdk.Info("Kubernetes monitor plugin initialized")

	// Get Kubernetes API config
	if url, ok := sdk.GetConfig("kubernetes_api_url"); ok {
		p.apiURL = url
	} else {
		p.apiURL = "https://kubernetes.default.svc"
	}

	if token, ok := sdk.GetConfig("kubernetes_token"); ok {
		p.token = token
	}

	if ns, ok := sdk.GetConfig("namespace"); ok {
		p.namespace = ns
	} else {
		p.namespace = "default"
	}

	sdk.Debug("K8s API URL: " + p.apiURL)
	return nil
}

func (p *KubernetesMonitorPlugin) Cleanup() error {
	sdk.Info("Kubernetes monitor plugin cleanup")
	return nil
}

func (p *KubernetesMonitorPlugin) OnTick() error {
	return p.CollectMetrics()
}

func (p *KubernetesMonitorPlugin) CollectMetrics() error {
	// Collect node metrics
	p.collectNodeMetrics()

	// Collect pod metrics
	p.collectPodMetrics()

	// Collect deployment metrics
	p.collectDeploymentMetrics()

	return nil
}

func (p *KubernetesMonitorPlugin) collectNodeMetrics() {
	// In a real plugin, we would call the K8s API
	// GET /api/v1/nodes
	sdk.Debug("Collecting node metrics")

	// Mock node metrics
	sdk.RecordMetricWithTags("k8s.node.count", 3, map[string]string{
		"cluster": "production",
	})

	sdk.RecordMetricWithTags("k8s.node.cpu.allocatable", 8000, map[string]string{
		"node": "node-1",
		"unit": "millicores",
	})

	sdk.RecordMetricWithTags("k8s.node.memory.allocatable", 32768, map[string]string{
		"node": "node-1",
		"unit": "MB",
	})
}

func (p *KubernetesMonitorPlugin) collectPodMetrics() {
	sdk.Debug("Collecting pod metrics")

	// Mock pod metrics
	sdk.RecordMetricWithTags("k8s.pod.count", 25, map[string]string{
		"namespace": p.namespace,
		"status":    "Running",
	})

	sdk.RecordMetricWithTags("k8s.pod.restarts", 2, map[string]string{
		"namespace": p.namespace,
		"pod":       "api-server-abc123",
	})
}

func (p *KubernetesMonitorPlugin) collectDeploymentMetrics() {
	sdk.Debug("Collecting deployment metrics")

	// Mock deployment metrics
	sdk.RecordMetricWithTags("k8s.deployment.replicas.desired", 3, map[string]string{
		"namespace":  p.namespace,
		"deployment": "api-server",
	})

	sdk.RecordMetricWithTags("k8s.deployment.replicas.ready", 3, map[string]string{
		"namespace":  p.namespace,
		"deployment": "api-server",
	})
}

func main() {
	sdk.Register(&KubernetesMonitorPlugin{})
}

