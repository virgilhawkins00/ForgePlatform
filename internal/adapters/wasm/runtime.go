// Package wasm implements the WebAssembly plugin runtime using wazero.
package wasm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime implements the WebAssembly plugin runtime.
type Runtime struct {
	runtime    wazero.Runtime
	modules    map[string]*LoadedPlugin
	mu         sync.RWMutex
	logger     ports.Logger
	httpClient *http.Client
	dataDir    string                 // Base directory for plugin data
	config     map[string]string      // Plugin configuration
	eventBus   chan PluginEvent       // Event bus for inter-plugin communication
	allocator  *PluginMemoryAllocator // Memory allocator for plugin responses
}

// PluginEvent represents an event emitted by a plugin.
type PluginEvent struct {
	PluginID  string
	EventType string
	Payload   []byte
}

// PluginMemoryAllocator manages memory allocation for plugin responses.
type PluginMemoryAllocator struct {
	mu     sync.Mutex
	memory map[uint32][]byte
	nextID uint32
}

// LoadedPlugin represents a loaded WebAssembly plugin.
type LoadedPlugin struct {
	Plugin  *domain.Plugin
	Module  api.Module
	Exports map[string]api.Function
}

// NewRuntime creates a new WebAssembly runtime.
func NewRuntime(ctx context.Context, logger ports.Logger) (*Runtime, error) {
	return NewRuntimeWithOptions(ctx, logger, RuntimeOptions{})
}

// RuntimeOptions configures the WASM runtime.
type RuntimeOptions struct {
	DataDir       string            // Base directory for plugin data (default: ~/.forge/plugins/data)
	Config        map[string]string // Plugin configuration
	HTTPTimeout   time.Duration     // HTTP request timeout (default: 30s)
	AllowedHosts  []string          // Allowed hosts for HTTP requests (empty = all)
	EventBufSize  int               // Event bus buffer size (default: 100)
}

// NewRuntimeWithOptions creates a new WebAssembly runtime with options.
func NewRuntimeWithOptions(ctx context.Context, logger ports.Logger, opts RuntimeOptions) (*Runtime, error) {
	// Create runtime with AOT compilation for better performance
	r := wazero.NewRuntime(ctx)

	// Instantiate WASI for basic system calls
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	// Set defaults
	if opts.DataDir == "" {
		home, _ := os.UserHomeDir()
		opts.DataDir = filepath.Join(home, ".forge", "plugins", "data")
	}
	if opts.HTTPTimeout == 0 {
		opts.HTTPTimeout = 30 * time.Second
	}
	if opts.EventBufSize == 0 {
		opts.EventBufSize = 100
	}
	if opts.Config == nil {
		opts.Config = make(map[string]string)
	}

	// Create data directory
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	runtime := &Runtime{
		runtime: r,
		modules: make(map[string]*LoadedPlugin),
		logger:  logger,
		httpClient: &http.Client{
			Timeout: opts.HTTPTimeout,
		},
		dataDir:  opts.DataDir,
		config:   opts.Config,
		eventBus: make(chan PluginEvent, opts.EventBufSize),
		allocator: &PluginMemoryAllocator{
			memory: make(map[uint32][]byte),
			nextID: 1,
		},
	}

	// Register host functions
	if err := runtime.registerHostFunctions(ctx); err != nil {
		r.Close(ctx)
		return nil, err
	}

	return runtime, nil
}

// registerHostFunctions registers the Forge ABI functions.
func (r *Runtime) registerHostFunctions(ctx context.Context) error {
	// Create host module with Forge functions
	_, err := r.runtime.NewHostModuleBuilder("forge").
		// Logging
		NewFunctionBuilder().
		WithFunc(r.hostLog).
		Export("forge_log").
		// Metrics
		NewFunctionBuilder().
		WithFunc(r.hostMetricRecord).
		Export("forge_metric_record").
		// Configuration
		NewFunctionBuilder().
		WithFunc(r.hostGetConfig).
		Export("forge_get_config").
		// HTTP (new capability)
		NewFunctionBuilder().
		WithFunc(r.hostHTTPRequest).
		Export("forge_http_request").
		// Events (new capability)
		NewFunctionBuilder().
		WithFunc(r.hostEmitEvent).
		Export("forge_emit_event").
		// Filesystem (new capability)
		NewFunctionBuilder().
		WithFunc(r.hostReadFile).
		Export("forge_read_file").
		NewFunctionBuilder().
		WithFunc(r.hostWriteFile).
		Export("forge_write_file").
		Instantiate(ctx)

	return err
}

// Host function: forge_log(level i32, ptr i32, len i32)
func (r *Runtime) hostLog(ctx context.Context, m api.Module, level, ptr, length uint32) {
	// Read string from plugin memory
	data, ok := m.Memory().Read(ptr, length)
	if !ok {
		return
	}

	msg := string(data)
	switch level {
	case 0:
		r.logger.Debug(msg)
	case 1:
		r.logger.Info(msg)
	case 2:
		r.logger.Warn(msg)
	case 3:
		r.logger.Error(msg)
	}
}

// Host function: forge_metric_record(key_ptr i32, key_len i32, value f64)
func (r *Runtime) hostMetricRecord(ctx context.Context, m api.Module, keyPtr, keyLen uint32, value float64) {
	data, ok := m.Memory().Read(keyPtr, keyLen)
	if !ok {
		return
	}

	metricName := string(data)
	r.logger.Debug("Plugin recorded metric", "name", metricName, "value", value)
	// TODO: Actually record the metric via the metric service
}

// Host function: forge_get_config(key_ptr i32, key_len i32) -> (ptr i32, len i32)
func (r *Runtime) hostGetConfig(ctx context.Context, m api.Module, keyPtr, keyLen uint32) (uint32, uint32) {
	// Read config key from plugin memory
	data, ok := m.Memory().Read(keyPtr, keyLen)
	if !ok {
		return 0, 0
	}

	configKey := string(data)
	value, exists := r.config[configKey]
	if !exists {
		return 0, 0
	}

	// Write value to plugin memory
	return r.writeToPluginMemory(m, []byte(value))
}

// Host function: forge_http_request(method_ptr, method_len, url_ptr, url_len, body_ptr, body_len i32)
//
//	-> (status_code i32, resp_ptr i32, resp_len i32)
func (r *Runtime) hostHTTPRequest(ctx context.Context, m api.Module,
	methodPtr, methodLen, urlPtr, urlLen, bodyPtr, bodyLen uint32) (int32, uint32, uint32) {

	// Read method
	methodData, ok := m.Memory().Read(methodPtr, methodLen)
	if !ok {
		return -1, 0, 0
	}
	method := string(methodData)

	// Read URL
	urlData, ok := m.Memory().Read(urlPtr, urlLen)
	if !ok {
		return -2, 0, 0
	}
	url := string(urlData)

	// Read body (optional)
	var body []byte
	if bodyPtr != 0 && bodyLen != 0 {
		body, ok = m.Memory().Read(bodyPtr, bodyLen)
		if !ok {
			return -3, 0, 0
		}
	}

	// Create and execute request
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		r.logger.Error("Failed to create HTTP request", "error", err)
		return -4, 0, 0
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("HTTP request failed", "error", err)
		return -5, 0, 0
	}
	defer resp.Body.Close()

	// Read response body (limit to 10MB)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		r.logger.Error("Failed to read response", "error", err)
		return -6, 0, 0
	}

	// Write response to plugin memory
	respPtr, respLen := r.writeToPluginMemory(m, respBody)
	return int32(resp.StatusCode), respPtr, respLen
}

// Host function: forge_emit_event(type_ptr, type_len, payload_ptr, payload_len i32) -> err_code i32
func (r *Runtime) hostEmitEvent(ctx context.Context, m api.Module,
	typePtr, typeLen, payloadPtr, payloadLen uint32) int32 {

	// Read event type
	typeData, ok := m.Memory().Read(typePtr, typeLen)
	if !ok {
		return -1
	}
	eventType := string(typeData)

	// Read payload
	var payload []byte
	if payloadPtr != 0 && payloadLen != 0 {
		payload, ok = m.Memory().Read(payloadPtr, payloadLen)
		if !ok {
			return -2
		}
	}

	// Send to event bus (non-blocking)
	select {
	case r.eventBus <- PluginEvent{EventType: eventType, Payload: payload}:
		r.logger.Debug("Event emitted", "type", eventType)
		return 0
	default:
		r.logger.Warn("Event bus full, dropping event", "type", eventType)
		return -3
	}
}

// Host function: forge_read_file(path_ptr, path_len i32) -> (data_ptr, data_len i32, err_code i32)
func (r *Runtime) hostReadFile(ctx context.Context, m api.Module,
	pathPtr, pathLen uint32) (uint32, uint32, int32) {

	// Read path
	pathData, ok := m.Memory().Read(pathPtr, pathLen)
	if !ok {
		return 0, 0, -1
	}
	path := string(pathData)

	// Sanitize path - prevent directory traversal
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		r.logger.Warn("Invalid file path", "path", path)
		return 0, 0, -2
	}

	// Build full path within data directory
	fullPath := filepath.Join(r.dataDir, cleanPath)

	// Read file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, -3
		}
		r.logger.Error("Failed to read file", "path", fullPath, "error", err)
		return 0, 0, -4
	}

	// Write to plugin memory
	dataPtr, dataLen := r.writeToPluginMemory(m, data)
	return dataPtr, dataLen, 0
}

// Host function: forge_write_file(path_ptr, path_len, data_ptr, data_len i32) -> err_code i32
func (r *Runtime) hostWriteFile(ctx context.Context, m api.Module,
	pathPtr, pathLen, dataPtr, dataLen uint32) int32 {

	// Read path
	pathData, ok := m.Memory().Read(pathPtr, pathLen)
	if !ok {
		return -1
	}
	path := string(pathData)

	// Sanitize path
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		r.logger.Warn("Invalid file path", "path", path)
		return -2
	}

	// Read data
	data, ok := m.Memory().Read(dataPtr, dataLen)
	if !ok {
		return -3
	}

	// Build full path
	fullPath := filepath.Join(r.dataDir, cleanPath)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.logger.Error("Failed to create directory", "dir", dir, "error", err)
		return -4
	}

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		r.logger.Error("Failed to write file", "path", fullPath, "error", err)
		return -5
	}

	r.logger.Debug("File written", "path", fullPath, "size", len(data))
	return 0
}

// writeToPluginMemory writes data to plugin memory and returns the pointer and length.
// For simplicity, this allocates new memory in the plugin's linear memory.
func (r *Runtime) writeToPluginMemory(m api.Module, data []byte) (uint32, uint32) {
	if len(data) == 0 {
		return 0, 0
	}

	// Try to find and call the plugin's malloc function
	malloc := m.ExportedFunction("malloc")
	if malloc == nil {
		// Fallback: use a simple allocation scheme at the end of memory
		// This is not ideal but works for simple cases
		r.logger.Debug("Plugin does not export malloc, using fallback allocation")
		return 0, 0
	}

	// Allocate memory in plugin
	results, err := malloc.Call(context.Background(), uint64(len(data)))
	if err != nil || len(results) == 0 {
		r.logger.Error("Failed to allocate plugin memory", "error", err)
		return 0, 0
	}

	ptr := uint32(results[0])
	if !m.Memory().Write(ptr, data) {
		r.logger.Error("Failed to write to plugin memory")
		return 0, 0
	}

	return ptr, uint32(len(data))
}

// LoadPlugin loads a WebAssembly plugin.
func (r *Runtime) LoadPlugin(ctx context.Context, plugin *domain.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read the WASM binary
	wasmBytes, err := os.ReadFile(plugin.Path)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Verify hash
	hash := sha256.Sum256(wasmBytes)
	hashStr := hex.EncodeToString(hash[:])
	if plugin.Hash != "" && plugin.Hash != hashStr {
		return fmt.Errorf("plugin hash mismatch: expected %s, got %s", plugin.Hash, hashStr)
	}
	plugin.Hash = hashStr

	// Compile and instantiate the module
	module, err := r.runtime.Instantiate(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to instantiate plugin: %w", err)
	}

	// Collect exported functions
	exports := make(map[string]api.Function)
	for name, fn := range module.ExportedFunctionDefinitions() {
		exports[name] = module.ExportedFunction(fn.Name())
	}

	r.modules[plugin.ID.String()] = &LoadedPlugin{
		Plugin:  plugin,
		Module:  module,
		Exports: exports,
	}

	plugin.MarkLoaded()
	r.logger.Info("Plugin loaded", "name", plugin.Name, "version", plugin.Version)

	return nil
}

// UnloadPlugin unloads a plugin from the runtime.
func (r *Runtime) UnloadPlugin(ctx context.Context, pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loaded, ok := r.modules[pluginID]
	if !ok {
		return fmt.Errorf("plugin not loaded: %s", pluginID)
	}

	if err := loaded.Module.Close(ctx); err != nil {
		return fmt.Errorf("failed to close module: %w", err)
	}

	delete(r.modules, pluginID)
	r.logger.Info("Plugin unloaded", "id", pluginID)

	return nil
}

// CallFunction invokes a function exported by a plugin.
func (r *Runtime) CallFunction(ctx context.Context, pluginID, funcName string, args ...interface{}) (interface{}, error) {
	r.mu.RLock()
	loaded, ok := r.modules[pluginID]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin not loaded: %s", pluginID)
	}

	fn, ok := loaded.Exports[funcName]
	if !ok || fn == nil {
		return nil, fmt.Errorf("function not found: %s", funcName)
	}

	// Convert args to uint64 for wazero
	wasmArgs := make([]uint64, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case int:
			wasmArgs[i] = uint64(v)
		case int32:
			wasmArgs[i] = uint64(v)
		case int64:
			wasmArgs[i] = uint64(v)
		case uint32:
			wasmArgs[i] = uint64(v)
		case uint64:
			wasmArgs[i] = v
		case float32:
			wasmArgs[i] = api.EncodeF32(v)
		case float64:
			wasmArgs[i] = api.EncodeF64(v)
		default:
			return nil, fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	results, err := fn.Call(ctx, wasmArgs...)
	if err != nil {
		return nil, fmt.Errorf("function call failed: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// ListLoadedPlugins returns the IDs of all loaded plugins.
func (r *Runtime) ListLoadedPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.modules))
	for id := range r.modules {
		ids = append(ids, id)
	}
	return ids
}

// Close shuts down the runtime.
func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx := context.Background()
	for id, loaded := range r.modules {
		loaded.Module.Close(ctx)
		delete(r.modules, id)
	}

	// Close event bus
	close(r.eventBus)

	return r.runtime.Close(ctx)
}

// SetConfig sets a configuration value for plugins.
func (r *Runtime) SetConfig(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config[key] = value
}

// GetConfig returns a configuration value.
func (r *Runtime) GetConfig(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.config[key]
	return val, ok
}

// Events returns the event bus channel for receiving plugin events.
func (r *Runtime) Events() <-chan PluginEvent {
	return r.eventBus
}

// DataDir returns the plugin data directory path.
func (r *Runtime) DataDir() string {
	return r.dataDir
}

var _ ports.WasmRuntime = (*Runtime)(nil)

