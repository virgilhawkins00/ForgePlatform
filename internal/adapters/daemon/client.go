package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Client is a client for communicating with the daemon.
type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	timeout    time.Duration
}

// NewClient creates a new daemon client.
func NewClient(forgeDir string) (*Client, error) {
	socketPath := filepath.Join(forgeDir, "forge.sock")

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("daemon not running (socket not found)")
	}

	return &Client{
		socketPath: socketPath,
		timeout:    120 * time.Second,
	}, nil
}

// Connect establishes a connection to the daemon.
func (c *Client) Connect() error {
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Call makes an RPC call to the daemon.
func (c *Client) Call(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	if c.conn == nil {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	// Create request
	req := Request{
		Method: method,
		Params: params,
		ID:     uuid.New().String(),
	}

	// Send request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	// Respect context deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(deadline)
	}

	if _, err := c.conn.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response with context deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetReadDeadline(deadline)
	} else {
		c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}

	// Convert result to map
	if result, ok := resp.Result.(map[string]interface{}); ok {
		return result, nil
	}

	return nil, nil
}

// Status gets the daemon status.
func (c *Client) Status(ctx context.Context) (map[string]interface{}, error) {
	return c.Call(ctx, "status", nil)
}

// RecordMetric records a metric.
func (c *Client) RecordMetric(ctx context.Context, name string, value float64, tags map[string]string) error {
	params := map[string]interface{}{
		"name":  name,
		"value": value,
		"tags":  tags,
	}

	_, err := c.Call(ctx, "metric.record", params)
	return err
}

// ListTasks lists tasks in the queue.
func (c *Client) ListTasks(ctx context.Context, status string) ([]interface{}, error) {
	params := map[string]interface{}{}
	if status != "" {
		params["status"] = status
	}

	resp, err := c.Call(ctx, "task.list", params)
	if err != nil {
		return nil, err
	}

	// Convert map to slice if the result is wrapped
	if tasks, ok := resp["tasks"].([]interface{}); ok {
		return tasks, nil
	}

	return nil, nil
}

// ListPlugins lists installed plugins.
func (c *Client) ListPlugins(ctx context.Context) ([]interface{}, error) {
	resp, err := c.Call(ctx, "plugin.list", nil)
	if err != nil {
		return nil, err
	}

	if plugins, ok := resp["plugins"].([]interface{}); ok {
		return plugins, nil
	}

	return nil, nil
}

// QueryMetric queries the latest values for a metric.
func (c *Client) QueryMetric(ctx context.Context, name string, limit int) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"name":  name,
		"limit": limit,
	}

	resp, err := c.Call(ctx, "metric.query", params)
	if err != nil {
		return nil, err
	}

	// Convert result to []map[string]interface{}
	var metrics []map[string]interface{}
	if points, ok := resp["points"].([]interface{}); ok {
		for _, p := range points {
			if m, ok := p.(map[string]interface{}); ok {
				metrics = append(metrics, m)
			}
		}
	}

	return metrics, nil
}

// GetMetricStats returns TSDB statistics.
func (c *Client) GetMetricStats(ctx context.Context) (map[string]interface{}, error) {
	return c.Call(ctx, "metric.stats", nil)
}
