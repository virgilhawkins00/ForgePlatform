// Package sdk provides the SDK for developing Forge WebAssembly plugins.
// This package is designed to be compiled with TinyGo to WebAssembly.
//
// # Quick Start
//
// Create a plugin by implementing the Plugin interface:
//
//	type MyPlugin struct{}
//
//	func (p *MyPlugin) Name() string    { return "my-plugin" }
//	func (p *MyPlugin) Version() string { return "1.0.0" }
//	func (p *MyPlugin) Init() error     { return nil }
//	func (p *MyPlugin) Cleanup() error  { return nil }
//
// Then register it in main():
//
//	func main() {
//	    sdk.Register(&MyPlugin{})
//	}
//
// Build with TinyGo:
//
//	tinygo build -o plugin.wasm -target=wasi main.go
package sdk

// LogLevel represents the severity of a log message.
type LogLevel int32

const (
	LogDebug LogLevel = 0
	LogInfo  LogLevel = 1
	LogWarn  LogLevel = 2
	LogError LogLevel = 3
)

// Plugin is the interface that all Forge plugins must implement.
type Plugin interface {
	// Name returns the plugin name.
	Name() string

	// Version returns the plugin version.
	Version() string

	// Init is called when the plugin is loaded.
	Init() error

	// Cleanup is called when the plugin is unloaded.
	Cleanup() error
}

// TickHandler is implemented by plugins that want periodic callbacks.
type TickHandler interface {
	// OnTick is called periodically by the Forge runtime.
	// The interval is configured in the plugin manifest.
	OnTick() error
}

// CommandHandler is implemented by plugins that handle CLI commands.
type CommandHandler interface {
	// HandleCommand is called when a command is invoked.
	// Returns the output string and any error.
	HandleCommand(command string, args []string) (string, error)
}

// MetricCollector is implemented by plugins that collect metrics.
type MetricCollector interface {
	// CollectMetrics is called to collect metrics.
	// Use RecordMetric() to record values.
	CollectMetrics() error
}

// EventHandler is implemented by plugins that handle events.
type EventHandler interface {
	// OnEvent is called when an event occurs.
	OnEvent(eventType string, payload []byte) error
}

// HTTPHandler is implemented by plugins that handle HTTP requests.
type HTTPHandler interface {
	// HandleHTTP is called for HTTP requests routed to this plugin.
	HandleHTTP(method, path string, body []byte) (statusCode int, response []byte, err error)
}

// ConfigProvider is implemented by plugins that need configuration.
type ConfigProvider interface {
	// ConfigSchema returns the JSON schema for plugin configuration.
	ConfigSchema() string

	// Configure is called with the plugin configuration.
	Configure(config []byte) error
}

// ========================================
// Host Function Imports (provided by Forge runtime)
// ========================================

// Host functions are declared in wasm_stubs.go (for testing) and
// wasm_imports.go (for actual WASM builds with TinyGo).
//
// Available host functions:
//   - forgeLog(level, ptr, length) - Write log message
//   - forgeMetricRecord(keyPtr, keyLen, value) - Record metric
//   - forgeGetConfig(keyPtr, keyLen) -> (ptr, length) - Get config value
//   - forgeHTTPRequest(...) -> (status, respPtr, respLen) - HTTP request
//   - forgeEmitEvent(...) -> errCode - Emit event
//   - forgeReadFile(pathPtr, pathLen) -> (dataPtr, dataLen, errCode) - Read file
//   - forgeWriteFile(pathPtr, pathLen, dataPtr, dataLen) -> errCode - Write file

// ========================================
// Logging Functions
// ========================================

// Log writes a log message at the specified level.
func Log(level LogLevel, message string) {
	ptr, length := stringToPtr(message)
	forgeLog(int32(level), ptr, length)
}

// Debug writes a debug log message.
func Debug(message string) {
	Log(LogDebug, message)
}

// Info writes an info log message.
func Info(message string) {
	Log(LogInfo, message)
}

// Warn writes a warning log message.
func Warn(message string) {
	Log(LogWarn, message)
}

// Error writes an error log message.
func Error(message string) {
	Log(LogError, message)
}

// ========================================
// Metric Functions
// ========================================

// RecordMetric records a metric value.
func RecordMetric(name string, value float64) {
	ptr, length := stringToPtr(name)
	forgeMetricRecord(ptr, length, value)
}

// RecordMetricWithTags records a metric with tags (encoded as name{tag=value}).
func RecordMetricWithTags(name string, value float64, tags map[string]string) {
	// Encode tags into metric name: name{key1=val1,key2=val2}
	if len(tags) > 0 {
		name = name + "{"
		first := true
		for k, v := range tags {
			if !first {
				name = name + ","
			}
			name = name + k + "=" + v
			first = false
		}
		name = name + "}"
	}
	RecordMetric(name, value)
}

// ========================================
// Configuration Functions
// ========================================

// GetConfig retrieves a configuration value by key.
func GetConfig(key string) (string, bool) {
	keyPtr, keyLen := stringToPtr(key)
	ptr, length := forgeGetConfig(keyPtr, keyLen)
	if ptr == 0 && length == 0 {
		return "", false
	}
	return ptrToString(ptr, length), true
}

// ========================================
// HTTP Functions (Sandboxed)
// ========================================

// HTTPResponse represents an HTTP response.
type HTTPResponse struct {
	StatusCode int
	Body       []byte
}

// HTTPGet performs a GET request to the specified URL.
func HTTPGet(url string) (*HTTPResponse, error) {
	return httpRequest("GET", url, nil)
}

// HTTPPost performs a POST request to the specified URL.
func HTTPPost(url string, body []byte) (*HTTPResponse, error) {
	return httpRequest("POST", url, body)
}

// HTTPPut performs a PUT request to the specified URL.
func HTTPPut(url string, body []byte) (*HTTPResponse, error) {
	return httpRequest("PUT", url, body)
}

// HTTPDelete performs a DELETE request to the specified URL.
func HTTPDelete(url string) (*HTTPResponse, error) {
	return httpRequest("DELETE", url, nil)
}

func httpRequest(method, url string, body []byte) (*HTTPResponse, error) {
	methodPtr, methodLen := stringToPtr(method)
	urlPtr, urlLen := stringToPtr(url)
	var bodyPtr, bodyLen uint32
	if body != nil {
		bodyPtr, bodyLen = bytesToPtr(body)
	}

	statusCode, respPtr, respLen := forgeHTTPRequest(methodPtr, methodLen, urlPtr, urlLen, bodyPtr, bodyLen)
	if statusCode < 0 {
		return nil, &PluginError{Code: int(statusCode), Message: "HTTP request failed"}
	}

	var respBody []byte
	if respPtr != 0 && respLen != 0 {
		respBody = ptrToBytes(respPtr, respLen)
	}

	return &HTTPResponse{
		StatusCode: int(statusCode),
		Body:       respBody,
	}, nil
}

// ========================================
// Event Functions
// ========================================

// EmitEvent emits an event that other plugins can subscribe to.
func EmitEvent(eventType string, payload []byte) error {
	typePtr, typeLen := stringToPtr(eventType)
	payloadPtr, payloadLen := bytesToPtr(payload)
	result := forgeEmitEvent(typePtr, typeLen, payloadPtr, payloadLen)
	if result != 0 {
		return &PluginError{Code: int(result), Message: "failed to emit event"}
	}
	return nil
}

// ========================================
// File System Functions (Scoped)
// ========================================

// ReadFile reads a file from the plugin's data directory.
func ReadFile(path string) ([]byte, error) {
	pathPtr, pathLen := stringToPtr(path)
	dataPtr, dataLen, errCode := forgeReadFile(pathPtr, pathLen)
	if errCode != 0 {
		return nil, &PluginError{Code: int(errCode), Message: "failed to read file"}
	}
	return ptrToBytes(dataPtr, dataLen), nil
}

// WriteFile writes data to a file in the plugin's data directory.
func WriteFile(path string, data []byte) error {
	pathPtr, pathLen := stringToPtr(path)
	dataPtr, dataLen := bytesToPtr(data)
	errCode := forgeWriteFile(pathPtr, pathLen, dataPtr, dataLen)
	if errCode != 0 {
		return &PluginError{Code: int(errCode), Message: "failed to write file"}
	}
	return nil
}

// ========================================
// Error Types
// ========================================

// PluginError represents an error from the Forge runtime.
type PluginError struct {
	Code    int
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}

// ========================================
// Memory Helpers (TinyGo compatible)
// ========================================
//
// Note: Memory helper implementations are in wasm_memory.go for WASM builds
// and wasm_stubs.go for non-WASM builds.
//
// These are forward declarations - implementations vary by build target:
//   - stringToPtr: Converts Go string to WASM linear memory pointer
//   - bytesToPtr: Converts Go []byte to WASM linear memory pointer
//   - ptrToString: Converts WASM pointer back to Go string
//   - ptrToBytes: Converts WASM pointer back to Go []byte

// ========================================
// Plugin Registration
// ========================================

var registeredPlugin Plugin

// Register registers a plugin with the Forge runtime.
// This should be called from main().
func Register(p Plugin) {
	registeredPlugin = p
}

// GetRegisteredPlugin returns the registered plugin.
func GetRegisteredPlugin() Plugin {
	return registeredPlugin
}

