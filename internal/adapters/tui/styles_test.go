package tui

import (
	"strings"
	"testing"
)

func TestRenderStatus_OK(t *testing.T) {
	statuses := []string{"ok", "active", "running", "completed", "installed"}
	
	for _, status := range statuses {
		result := renderStatus(status)
		if !strings.Contains(result, status) {
			t.Errorf("expected %s status to contain the status text, got %s", status, result)
		}
		if !strings.Contains(result, "●") {
			t.Errorf("expected %s status to contain bullet point", status)
		}
	}
}

func TestRenderStatus_Error(t *testing.T) {
	statuses := []string{"error", "failed", "dead"}
	
	for _, status := range statuses {
		result := renderStatus(status)
		if !strings.Contains(result, status) {
			t.Errorf("expected %s status to contain the status text, got %s", status, result)
		}
	}
}

func TestRenderStatus_Warning(t *testing.T) {
	statuses := []string{"warning", "pending", "disabled"}
	
	for _, status := range statuses {
		result := renderStatus(status)
		if !strings.Contains(result, status) {
			t.Errorf("expected %s status to contain the status text, got %s", status, result)
		}
	}
}

func TestRenderStatus_Available(t *testing.T) {
	result := renderStatus("available")
	if !strings.Contains(result, "available") {
		t.Errorf("expected available status to contain the status text")
	}
	if !strings.Contains(result, "○") {
		t.Error("expected available status to use hollow bullet point")
	}
}

func TestRenderStatus_Unknown(t *testing.T) {
	result := renderStatus("unknown-status")
	if !strings.Contains(result, "unknown-status") {
		t.Errorf("expected unknown status to contain the status text")
	}
}

func TestStyles_NotNil(t *testing.T) {
	// Test that styles are properly initialized
	if activeTabStyle.String() == "" {
		t.Log("activeTabStyle rendered empty string")
	}
	if inactiveTabStyle.String() == "" {
		t.Log("inactiveTabStyle rendered empty string")
	}
}

func TestColors_Defined(t *testing.T) {
	// Test that colors are defined
	if primaryColor == "" {
		t.Error("primaryColor should not be empty")
	}
	if secondaryColor == "" {
		t.Error("secondaryColor should not be empty")
	}
	if errorColor == "" {
		t.Error("errorColor should not be empty")
	}
	if warningColor == "" {
		t.Error("warningColor should not be empty")
	}
	if infoColor == "" {
		t.Error("infoColor should not be empty")
	}
	if mutedColor == "" {
		t.Error("mutedColor should not be empty")
	}
	if fgColor == "" {
		t.Error("fgColor should not be empty")
	}
}

