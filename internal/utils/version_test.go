package utils

import (
	"testing"
)

func TestParseImageVersion_ValidCETag(t *testing.T) {
	major, minor, patch, err := ParseImageVersion("aerospike:ce-8.1.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if major != 8 || minor != 1 || patch != 1 {
		t.Errorf("got %d.%d.%d, want 8.1.1", major, minor, patch)
	}
}

func TestParseImageVersion_ValidCE7Tag(t *testing.T) {
	major, minor, patch, err := ParseImageVersion("aerospike:ce-7.2.0.6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if major != 7 || minor != 2 || patch != 0 {
		t.Errorf("got %d.%d.%d, want 7.2.0", major, minor, patch)
	}
}

func TestParseImageVersion_NoTag(t *testing.T) {
	_, _, _, err := ParseImageVersion("aerospike")
	if err == nil {
		t.Error("expected error for image without tag")
	}
}

func TestParseImageVersion_EmptyTag(t *testing.T) {
	_, _, _, err := ParseImageVersion("aerospike:")
	if err == nil {
		t.Error("expected error for empty tag")
	}
}

func TestParseImageVersion_InvalidFormat(t *testing.T) {
	_, _, _, err := ParseImageVersion("aerospike:latest")
	if err == nil {
		t.Error("expected error for non-semver tag")
	}
}

func TestParseImageVersion_EETag(t *testing.T) {
	major, minor, patch, err := ParseImageVersion("aerospike:ee-7.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if major != 7 || minor != 1 || patch != 0 {
		t.Errorf("got %d.%d.%d, want 7.1.0", major, minor, patch)
	}
}

func TestParseImageVersion_WithSuffix(t *testing.T) {
	major, minor, patch, err := ParseImageVersion("aerospike:ce-8.0.0-rc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if major != 8 || minor != 0 || patch != 0 {
		t.Errorf("got %d.%d.%d, want 8.0.0", major, minor, patch)
	}
}

func TestIsEnterpriseImage_CEImage(t *testing.T) {
	if IsEnterpriseImage("aerospike:ce-8.1.1.1") {
		t.Error("CE image should not be enterprise")
	}
}

func TestIsEnterpriseImage_EETag(t *testing.T) {
	if !IsEnterpriseImage("aerospike:ee-7.1.0") {
		t.Error("ee- tag should be enterprise")
	}
}

func TestIsEnterpriseImage_EnterpriseInName(t *testing.T) {
	if !IsEnterpriseImage("aerospike/aerospike-server-enterprise:7.1.0") {
		t.Error("enterprise in image name should be enterprise")
	}
}

func TestIsEnterpriseImage_PlainImage(t *testing.T) {
	if IsEnterpriseImage("aerospike:7.1.0") {
		t.Error("plain image should not be enterprise")
	}
}
