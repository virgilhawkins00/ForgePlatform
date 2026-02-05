//go:build !tinygo.wasm

// Package sdk provides stubs for non-WASM builds.
// These functions are only used when testing the SDK outside of WASM.
package sdk

import "unsafe"

// Stub implementations for non-WASM builds.
// These are replaced by actual WASM imports when compiled with TinyGo.

func forgeLog(level int32, ptr, length uint32) {
	// Stub - no-op in non-WASM builds
}

func forgeMetricRecord(keyPtr, keyLen uint32, value float64) {
	// Stub - no-op in non-WASM builds
}

func forgeGetConfig(keyPtr, keyLen uint32) (ptr, length uint32) {
	// Stub - returns empty in non-WASM builds
	return 0, 0
}

func forgeHTTPRequest(methodPtr, methodLen, urlPtr, urlLen, bodyPtr, bodyLen uint32) (statusCode int32, respPtr, respLen uint32) {
	// Stub - returns error in non-WASM builds
	return -1, 0, 0
}

func forgeEmitEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) int32 {
	// Stub - returns error in non-WASM builds
	return -1
}

func forgeReadFile(pathPtr, pathLen uint32) (dataPtr, dataLen uint32, errCode int32) {
	// Stub - returns error in non-WASM builds
	return 0, 0, -1
}

func forgeWriteFile(pathPtr, pathLen, dataPtr, dataLen uint32) int32 {
	// Stub - returns error in non-WASM builds
	return -1
}

// ========================================
// Memory Helpers (stub implementations)
// ========================================

// stringToPtr converts a string to pointer and length.
// In stub mode, this returns 0,0 since we're not in WASM.
func stringToPtr(s string) (uint32, uint32) {
	if len(s) == 0 {
		return 0, 0
	}
	// For non-WASM builds, we can use unsafe to get real pointers
	// but they won't work with host functions (which are stubs anyway)
	ptr := unsafe.Pointer(unsafe.StringData(s))
	return uint32(uintptr(ptr)), uint32(len(s))
}

// bytesToPtr converts a byte slice to pointer and length.
func bytesToPtr(b []byte) (uint32, uint32) {
	if len(b) == 0 {
		return 0, 0
	}
	ptr := unsafe.Pointer(&b[0])
	return uint32(uintptr(ptr)), uint32(len(b))
}

// ptrToString converts a pointer and length to a string.
// In stub mode, this attempts to read from the pointer.
func ptrToString(ptr, length uint32) string {
	if ptr == 0 || length == 0 {
		return ""
	}
	// In non-WASM builds, treat ptr as actual memory address
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	return string(bytes)
}

// ptrToBytes converts a pointer and length to a byte slice.
func ptrToBytes(ptr, length uint32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	result := make([]byte, length)
	copy(result, bytes)
	return result
}

