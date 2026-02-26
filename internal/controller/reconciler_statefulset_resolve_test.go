package controller

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestResolveIntOrPercent_Nil(t *testing.T) {
	if got := resolveIntOrPercent(nil, 10); got != 1 {
		t.Errorf("resolveIntOrPercent(nil, 10) = %d, want 1", got)
	}
}

func TestResolveIntOrPercent_IntValue(t *testing.T) {
	val := intstr.FromInt32(3)
	if got := resolveIntOrPercent(&val, 10); got != 3 {
		t.Errorf("resolveIntOrPercent(3, 10) = %d, want 3", got)
	}
}

func TestResolveIntOrPercent_IntValueZero(t *testing.T) {
	val := intstr.FromInt32(0)
	if got := resolveIntOrPercent(&val, 10); got != 1 {
		t.Errorf("resolveIntOrPercent(0, 10) = %d, want 1 (minimum)", got)
	}
}

func TestResolveIntOrPercent_IntValueNegative(t *testing.T) {
	val := intstr.FromInt32(-1)
	if got := resolveIntOrPercent(&val, 10); got != 1 {
		t.Errorf("resolveIntOrPercent(-1, 10) = %d, want 1 (minimum)", got)
	}
}

func TestResolveIntOrPercent_Percent50(t *testing.T) {
	val := intstr.FromString("50%")
	got := resolveIntOrPercent(&val, 10)
	if got != 5 {
		t.Errorf("resolveIntOrPercent(50%%, 10) = %d, want 5", got)
	}
}

func TestResolveIntOrPercent_Percent10_RoundsUp(t *testing.T) {
	val := intstr.FromString("10%")
	got := resolveIntOrPercent(&val, 3)
	// 10% of 3 = 0.3, rounds up to 1
	if got != 1 {
		t.Errorf("resolveIntOrPercent(10%%, 3) = %d, want 1 (rounds up)", got)
	}
}

func TestResolveIntOrPercent_Percent100(t *testing.T) {
	val := intstr.FromString("100%")
	got := resolveIntOrPercent(&val, 6)
	if got != 6 {
		t.Errorf("resolveIntOrPercent(100%%, 6) = %d, want 6", got)
	}
}

func TestResolveIntOrPercent_InvalidPercentString(t *testing.T) {
	val := intstr.FromString("abc")
	got := resolveIntOrPercent(&val, 10)
	if got != 1 {
		t.Errorf("resolveIntOrPercent(abc, 10) = %d, want 1 (fallback)", got)
	}
}
