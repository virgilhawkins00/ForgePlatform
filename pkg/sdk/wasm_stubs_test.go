//go:build !tinygo.wasm

package sdk

import (
	"testing"
)

func TestStringToPtr(t *testing.T) {
	// Empty string
	ptr, length := stringToPtr("")
	if ptr != 0 || length != 0 {
		t.Errorf("empty string should return 0,0, got %d,%d", ptr, length)
	}

	// Non-empty string
	ptr, length = stringToPtr("hello")
	if ptr == 0 {
		t.Error("expected non-zero pointer for non-empty string")
	}
	if length != 5 {
		t.Errorf("expected length 5, got %d", length)
	}
}

func TestBytesToPtr(t *testing.T) {
	// Empty slice
	ptr, length := bytesToPtr(nil)
	if ptr != 0 || length != 0 {
		t.Errorf("nil slice should return 0,0, got %d,%d", ptr, length)
	}

	ptr, length = bytesToPtr([]byte{})
	if ptr != 0 || length != 0 {
		t.Errorf("empty slice should return 0,0, got %d,%d", ptr, length)
	}

	// Non-empty slice
	ptr, length = bytesToPtr([]byte{1, 2, 3, 4, 5})
	if ptr == 0 {
		t.Error("expected non-zero pointer for non-empty slice")
	}
	if length != 5 {
		t.Errorf("expected length 5, got %d", length)
	}
}

func TestPtrToString(t *testing.T) {
	// Zero pointer
	s := ptrToString(0, 0)
	if s != "" {
		t.Errorf("expected empty string for 0 pointer, got %q", s)
	}

	s = ptrToString(0, 10)
	if s != "" {
		t.Errorf("expected empty string for 0 pointer, got %q", s)
	}

	// Valid pointer
	orig := "hello world"
	ptr, length := stringToPtr(orig)
	result := ptrToString(ptr, length)
	if result != orig {
		t.Errorf("expected %q, got %q", orig, result)
	}
}

func TestPtrToBytes(t *testing.T) {
	// Zero pointer
	b := ptrToBytes(0, 0)
	if b != nil {
		t.Errorf("expected nil for 0 pointer, got %v", b)
	}

	b = ptrToBytes(0, 10)
	if b != nil {
		t.Errorf("expected nil for 0 pointer, got %v", b)
	}

	// Valid pointer
	orig := []byte{1, 2, 3, 4, 5}
	ptr, length := bytesToPtr(orig)
	result := ptrToBytes(ptr, length)
	if len(result) != len(orig) {
		t.Errorf("expected length %d, got %d", len(orig), len(result))
	}
	for i, v := range result {
		if v != orig[i] {
			t.Errorf("byte %d: expected %d, got %d", i, orig[i], v)
		}
	}
}

func TestForgeLog_Stub(t *testing.T) {
	// Stub should not panic
	forgeLog(0, 0, 0)
	forgeLog(1, 100, 10)
}

func TestForgeMetricRecord_Stub(t *testing.T) {
	// Stub should not panic
	forgeMetricRecord(0, 0, 0)
	forgeMetricRecord(100, 10, 99.9)
}

func TestForgeGetConfig_Stub(t *testing.T) {
	ptr, length := forgeGetConfig(0, 0)
	if ptr != 0 || length != 0 {
		t.Errorf("expected 0,0 from stub, got %d,%d", ptr, length)
	}
}

func TestForgeHTTPRequest_Stub(t *testing.T) {
	status, respPtr, respLen := forgeHTTPRequest(0, 0, 0, 0, 0, 0)
	if status != -1 {
		t.Errorf("expected status -1 from stub, got %d", status)
	}
	if respPtr != 0 || respLen != 0 {
		t.Errorf("expected 0,0 response from stub, got %d,%d", respPtr, respLen)
	}
}

func TestForgeEmitEvent_Stub(t *testing.T) {
	result := forgeEmitEvent(0, 0, 0, 0)
	if result != -1 {
		t.Errorf("expected -1 from stub, got %d", result)
	}
}

func TestForgeReadFile_Stub(t *testing.T) {
	dataPtr, dataLen, errCode := forgeReadFile(0, 0)
	if dataPtr != 0 || dataLen != 0 {
		t.Errorf("expected 0,0 data from stub, got %d,%d", dataPtr, dataLen)
	}
	if errCode != -1 {
		t.Errorf("expected errCode -1 from stub, got %d", errCode)
	}
}

func TestForgeWriteFile_Stub(t *testing.T) {
	result := forgeWriteFile(0, 0, 0, 0)
	if result != -1 {
		t.Errorf("expected -1 from stub, got %d", result)
	}
}

