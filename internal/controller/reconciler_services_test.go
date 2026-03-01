package controller

import (
	"maps"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestServicePortsChanged(t *testing.T) {
	tests := []struct {
		name     string
		existing []corev1.ServicePort
		desired  []corev1.ServicePort
		want     bool
	}{
		{
			name:     "both empty",
			existing: nil,
			desired:  nil,
			want:     false,
		},
		{
			name:     "same single port",
			existing: []corev1.ServicePort{{Name: "service", Port: 3000}},
			desired:  []corev1.ServicePort{{Name: "service", Port: 3000}},
			want:     false,
		},
		{
			name:     "different lengths",
			existing: []corev1.ServicePort{{Name: "service", Port: 3000}},
			desired: []corev1.ServicePort{
				{Name: "service", Port: 3000},
				{Name: "fabric", Port: 3001},
			},
			want: true,
		},
		{
			name:     "different port name",
			existing: []corev1.ServicePort{{Name: "service", Port: 3000}},
			desired:  []corev1.ServicePort{{Name: "fabric", Port: 3000}},
			want:     true,
		},
		{
			name:     "different port number",
			existing: []corev1.ServicePort{{Name: "service", Port: 3000}},
			desired:  []corev1.ServicePort{{Name: "service", Port: 4000}},
			want:     true,
		},
		{
			name: "same multiple ports",
			existing: []corev1.ServicePort{
				{Name: "service", Port: 3000},
				{Name: "fabric", Port: 3001},
			},
			desired: []corev1.ServicePort{
				{Name: "service", Port: 3000},
				{Name: "fabric", Port: 3001},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := servicePortsChanged(tc.existing, tc.desired)
			if got != tc.want {
				t.Errorf("servicePortsChanged() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEqualAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]string
		desired  map[string]string
		expected bool
	}{
		{
			name:     "both nil",
			actual:   nil,
			desired:  nil,
			expected: true,
		},
		{
			name:     "desired nil, actual empty",
			actual:   map[string]string{},
			desired:  nil,
			expected: true,
		},
		{
			name:     "actual nil, desired empty map",
			actual:   nil,
			desired:  map[string]string{},
			expected: true,
		},
		{
			name:     "desired nil, actual has system annotation only",
			actual:   map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"},
			desired:  nil,
			expected: true,
		},
		{
			name:     "desired matches actual",
			actual:   map[string]string{"foo": "bar"},
			desired:  map[string]string{"foo": "bar"},
			expected: true,
		},
		{
			name:     "desired matches actual with system annotation preserved",
			actual:   map[string]string{"foo": "bar", "kubectl.kubernetes.io/last-applied-configuration": "{}"},
			desired:  map[string]string{"foo": "bar"},
			expected: true,
		},
		{
			name:     "desired adds new annotation",
			actual:   map[string]string{"foo": "bar"},
			desired:  map[string]string{"foo": "bar", "baz": "qux"},
			expected: false,
		},
		{
			name:     "desired updates value",
			actual:   map[string]string{"foo": "bar"},
			desired:  map[string]string{"foo": "new-value"},
			expected: false,
		},
		{
			name:     "desired removes annotation",
			actual:   map[string]string{"foo": "bar", "old": "stale"},
			desired:  map[string]string{"foo": "bar"},
			expected: false,
		},
		{
			name:     "desired removes all annotations",
			actual:   map[string]string{"foo": "bar"},
			desired:  nil,
			expected: false,
		},
		{
			name:     "desired nil, actual has system + operator annotation",
			actual:   map[string]string{"foo": "bar", "kubectl.kubernetes.io/last-applied-configuration": "{}"},
			desired:  nil,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := equalAnnotations(tc.actual, tc.desired)
			if got != tc.expected {
				t.Errorf("equalAnnotations(%v, %v) = %v, want %v", tc.actual, tc.desired, got, tc.expected)
			}
		})
	}
}

func TestReconcileAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]string
		desired  map[string]string
		expected map[string]string
	}{
		{
			name:     "both nil",
			existing: nil,
			desired:  nil,
			expected: nil,
		},
		{
			name:     "desired only",
			existing: nil,
			desired:  map[string]string{"foo": "bar"},
			expected: map[string]string{"foo": "bar"},
		},
		{
			name:     "existing only (non-system) — cleaned up",
			existing: map[string]string{"foo": "bar"},
			desired:  nil,
			expected: nil,
		},
		{
			name:     "existing empty map, desired nil",
			existing: map[string]string{},
			desired:  nil,
			expected: nil,
		},
		{
			name:     "existing system annotation preserved when desired is nil",
			existing: map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"},
			desired:  nil,
			expected: map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"},
		},
		{
			name:     "merge preserves system, applies desired",
			existing: map[string]string{"old": "stale", "kubectl.kubernetes.io/last-applied-configuration": "{}"},
			desired:  map[string]string{"new": "value"},
			expected: map[string]string{"new": "value", "kubectl.kubernetes.io/last-applied-configuration": "{}"},
		},
		{
			name:     "desired updates existing value",
			existing: map[string]string{"foo": "old"},
			desired:  map[string]string{"foo": "new"},
			expected: map[string]string{"foo": "new"},
		},
		{
			name:     "k8s.io system annotations preserved",
			existing: map[string]string{"app.k8s.io/managed-by": "helm", "user-key": "val"},
			desired:  map[string]string{"new-key": "new-val"},
			expected: map[string]string{"app.k8s.io/managed-by": "helm", "new-key": "new-val"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := reconcileAnnotations(tc.existing, tc.desired)
			if !maps.Equal(got, tc.expected) {
				t.Errorf("reconcileAnnotations(%v, %v) = %v, want %v", tc.existing, tc.desired, got, tc.expected)
			}
		})
	}
}

func TestIsSystemAnnotation(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"kubectl.kubernetes.io/last-applied-configuration", true},
		{"kubernetes.io/ingress-class", true},
		{"app.kubernetes.io/managed-by", true},
		{"app.k8s.io/managed-by", true},
		{"k8s.io/some-key", true},
		{"example.com/env", false},
		{"prometheus.io/scrape", false},
		{"acko.io/managed", false},
		{"foo", false},
		// Edge cases: substring matching should NOT match these
		{"bypass-kubernetes.io/hack", false},
		{"notkubernetes.io/key", false},
		{"notk8s.io/key", false},
		{"fakek8s.io/something", false},
		// No domain prefix (bare key)
		{"kubernetes.io", false},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := isSystemAnnotation(tc.key)
			if got != tc.expected {
				t.Errorf("isSystemAnnotation(%q) = %v, want %v", tc.key, got, tc.expected)
			}
		})
	}
}
