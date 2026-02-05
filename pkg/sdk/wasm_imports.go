//go:build tinygo.wasm

// Package sdk provides WASM imports for TinyGo builds.
// These functions import the host functions provided by the Forge runtime.
package sdk

import "unsafe"

// ========================================
// Host Function Imports
// ========================================

// forgeLog writes a log message to the Forge runtime.
//
//go:wasmimport forge forge_log
func forgeLog(level int32, ptr, length uint32)

// forgeMetricRecord records a metric value.
//
//go:wasmimport forge forge_metric_record
func forgeMetricRecord(keyPtr, keyLen uint32, value float64)

// forgeGetConfig retrieves a configuration value.
//
//go:wasmimport forge forge_get_config
func forgeGetConfig(keyPtr, keyLen uint32) (ptr, length uint32)

// forgeHTTPRequest performs an HTTP request.
//
//go:wasmimport forge forge_http_request
func forgeHTTPRequest(methodPtr, methodLen, urlPtr, urlLen, bodyPtr, bodyLen uint32) (statusCode int32, respPtr, respLen uint32)

// forgeEmitEvent emits an event to the event bus.
//
//go:wasmimport forge forge_emit_event
func forgeEmitEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) int32

// forgeReadFile reads a file from the plugin's data directory.
//
//go:wasmimport forge forge_read_file
func forgeReadFile(pathPtr, pathLen uint32) (dataPtr, dataLen uint32, errCode int32)

// forgeWriteFile writes a file to the plugin's data directory.
//
//go:wasmimport forge forge_write_file
func forgeWriteFile(pathPtr, pathLen, dataPtr, dataLen uint32) int32

// ========================================
// Memory Helpers (TinyGo WASM)
// ========================================

// stringToPtr converts a Go string to WASM linear memory pointer.
func stringToPtr(s string) (uint32, uint32) {
	if len(s) == 0 {
		return 0, 0
	}
	ptr := unsafe.Pointer(unsafe.StringData(s))
	return uint32(uintptr(ptr)), uint32(len(s))
}

// bytesToPtr converts a Go []byte to WASM linear memory pointer.
func bytesToPtr(b []byte) (uint32, uint32) {
	if len(b) == 0 {
		return 0, 0
	}
	ptr := unsafe.Pointer(&b[0])
	return uint32(uintptr(ptr)), uint32(len(b))
}

// ptrToString reads a string from WASM linear memory.
func ptrToString(ptr, length uint32) string {
	if ptr == 0 || length == 0 {
		return ""
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	return string(bytes)
}

// ptrToBytes reads a byte slice from WASM linear memory.
func ptrToBytes(ptr, length uint32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	result := make([]byte, length)
	copy(result, bytes)
	return result
}

