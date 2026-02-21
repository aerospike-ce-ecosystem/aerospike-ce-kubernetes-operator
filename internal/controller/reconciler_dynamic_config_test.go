package controller

import (
	"testing"

	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/configdiff"
)

// --- buildSetConfigCommand tests ---

func TestBuildSetConfigCommand_ServiceContext(t *testing.T) {
	change := configdiff.Change{
		Path:     "service.proto-fd-max",
		Context:  "service",
		Key:      "proto-fd-max",
		NewValue: 20000,
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=service;proto-fd-max=20000"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}

func TestBuildSetConfigCommand_NamespaceContext(t *testing.T) {
	change := configdiff.Change{
		Path:      "namespace.default-ttl",
		Context:   "namespace",
		Key:       "default-ttl",
		NewValue:  3600,
		Namespace: "myns",
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=namespace;id=myns;default-ttl=3600"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}

func TestBuildSetConfigCommand_NetworkContext(t *testing.T) {
	change := configdiff.Change{
		Path:     "network.heartbeat.interval",
		Context:  "network",
		Key:      "heartbeat.interval",
		NewValue: 250,
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=network;heartbeat.interval=250"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}

func TestBuildSetConfigCommand_StringValue(t *testing.T) {
	change := configdiff.Change{
		Path:     "service.ticker-interval",
		Context:  "service",
		Key:      "ticker-interval",
		NewValue: "10",
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=service;ticker-interval=10"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}

func TestBuildSetConfigCommand_BoolValue(t *testing.T) {
	change := configdiff.Change{
		Path:      "namespace.read-page-cache",
		Context:   "namespace",
		Key:       "read-page-cache",
		NewValue:  true,
		Namespace: "testns",
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=namespace;id=testns;read-page-cache=true"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}

func TestBuildSetConfigCommand_NamespaceWithHighWaterDiskPct(t *testing.T) {
	change := configdiff.Change{
		Path:      "namespace.high-water-disk-pct",
		Context:   "namespace",
		Key:       "high-water-disk-pct",
		NewValue:  90,
		Namespace: "production",
	}

	cmd := buildSetConfigCommand(change)
	expected := "set-config:context=namespace;id=production;high-water-disk-pct=90"
	if cmd != expected {
		t.Errorf("buildSetConfigCommand = %q, want %q", cmd, expected)
	}
}
