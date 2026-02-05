// Example Forge plugin: Cloud Metrics Collector
//
// This plugin demonstrates how to collect metrics from cloud providers.
// Supports AWS, GCP, and Azure with configurable regions.
//
// Build with TinyGo:
//   tinygo build -o cloud-metrics.wasm -target=wasi -scheduler=none .
//
// Install:
//   forge plugin install ./cloud-metrics.wasm
//
// Configuration:
//   cloud_provider: aws|gcp|azure
//   cloud_region: us-east-1
//   aws_access_key_id: <access-key>
//   aws_secret_access_key: <secret-key>

package main

import (
	"github.com/forge-platform/forge/pkg/sdk"
)

// CloudMetricsPlugin collects metrics from cloud providers.
type CloudMetricsPlugin struct {
	provider string
	region   string
}

var (
	_ sdk.Plugin          = (*CloudMetricsPlugin)(nil)
	_ sdk.TickHandler     = (*CloudMetricsPlugin)(nil)
	_ sdk.MetricCollector = (*CloudMetricsPlugin)(nil)
)

func (p *CloudMetricsPlugin) Name() string {
	return "cloud-metrics"
}

func (p *CloudMetricsPlugin) Version() string {
	return "1.0.0"
}

func (p *CloudMetricsPlugin) Init() error {
	sdk.Info("Cloud metrics plugin initialized")

	// Get cloud provider config
	if provider, ok := sdk.GetConfig("cloud_provider"); ok {
		p.provider = provider
	} else {
		p.provider = "aws"
	}

	if region, ok := sdk.GetConfig("cloud_region"); ok {
		p.region = region
	} else {
		p.region = "us-east-1"
	}

	sdk.Info("Cloud provider: " + p.provider + ", region: " + p.region)
	return nil
}

func (p *CloudMetricsPlugin) Cleanup() error {
	sdk.Info("Cloud metrics plugin cleanup")
	return nil
}

func (p *CloudMetricsPlugin) OnTick() error {
	return p.CollectMetrics()
}

func (p *CloudMetricsPlugin) CollectMetrics() error {
	switch p.provider {
	case "aws":
		return p.collectAWSMetrics()
	case "gcp":
		return p.collectGCPMetrics()
	case "azure":
		return p.collectAzureMetrics()
	default:
		sdk.Warn("Unknown cloud provider: " + p.provider)
		return nil
	}
}

func (p *CloudMetricsPlugin) collectAWSMetrics() error {
	sdk.Debug("Collecting AWS metrics")

	// EC2 instance metrics
	sdk.RecordMetricWithTags("cloud.ec2.instances.running", 15, map[string]string{
		"provider": "aws",
		"region":   p.region,
	})

	// RDS metrics
	sdk.RecordMetricWithTags("cloud.rds.connections", 42, map[string]string{
		"provider": "aws",
		"region":   p.region,
		"instance": "prod-db",
	})

	// S3 bucket metrics
	sdk.RecordMetricWithTags("cloud.s3.bucket.size.bytes", 1073741824, map[string]string{
		"provider": "aws",
		"region":   p.region,
		"bucket":   "my-app-assets",
	})

	// Lambda metrics
	sdk.RecordMetricWithTags("cloud.lambda.invocations", 1500, map[string]string{
		"provider": "aws",
		"region":   p.region,
		"function": "api-handler",
	})

	return nil
}

func (p *CloudMetricsPlugin) collectGCPMetrics() error {
	sdk.Debug("Collecting GCP metrics")

	// Compute Engine metrics
	sdk.RecordMetricWithTags("cloud.gce.instances.running", 10, map[string]string{
		"provider": "gcp",
		"region":   p.region,
	})

	// Cloud SQL metrics
	sdk.RecordMetricWithTags("cloud.cloudsql.connections", 28, map[string]string{
		"provider": "gcp",
		"region":   p.region,
		"instance": "prod-db",
	})

	return nil
}

func (p *CloudMetricsPlugin) collectAzureMetrics() error {
	sdk.Debug("Collecting Azure metrics")

	// Azure VMs
	sdk.RecordMetricWithTags("cloud.azure.vm.running", 8, map[string]string{
		"provider": "azure",
		"region":   p.region,
	})

	// Azure SQL
	sdk.RecordMetricWithTags("cloud.azure.sql.dtu.percent", 45.5, map[string]string{
		"provider": "azure",
		"region":   p.region,
		"database": "prod-db",
	})

	return nil
}

func main() {
	sdk.Register(&CloudMetricsPlugin{})
}

