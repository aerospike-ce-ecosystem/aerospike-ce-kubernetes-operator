package controller

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIntOrStringEqual_SameIntValues(t *testing.T) {
	a := intstr.FromInt32(1)
	b := intstr.FromInt32(1)
	if !intOrStringEqual(a, b) {
		t.Error("expected equal for same int values")
	}
}

func TestIntOrStringEqual_DifferentIntValues(t *testing.T) {
	a := intstr.FromInt32(1)
	b := intstr.FromInt32(2)
	if intOrStringEqual(a, b) {
		t.Error("expected not equal for different int values")
	}
}

func TestIntOrStringEqual_SameStringValues(t *testing.T) {
	a := intstr.FromString("50%")
	b := intstr.FromString("50%")
	if !intOrStringEqual(a, b) {
		t.Error("expected equal for same string values")
	}
}

func TestIntOrStringEqual_DifferentStringValues(t *testing.T) {
	a := intstr.FromString("50%")
	b := intstr.FromString("25%")
	if intOrStringEqual(a, b) {
		t.Error("expected not equal for different string values")
	}
}

func TestIntOrStringEqual_IntVsString(t *testing.T) {
	// Int(1) and String("1") should NOT be equal even though .String() would match
	a := intstr.FromInt32(1)
	b := intstr.FromString("1")
	if intOrStringEqual(a, b) {
		t.Error("expected not equal for int vs string with same textual representation")
	}
}

func TestIntOrStringEqual_ZeroValues(t *testing.T) {
	a := intstr.IntOrString{}
	b := intstr.IntOrString{}
	if !intOrStringEqual(a, b) {
		t.Error("expected equal for zero values")
	}
}
