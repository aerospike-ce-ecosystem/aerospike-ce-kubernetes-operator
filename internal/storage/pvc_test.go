package storage

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pvcWithName(name string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

// --- extractOrdinal tests ---

func TestExtractOrdinal_ValidPattern(t *testing.T) {
	// PVC name: <volumeName>-<stsName>-<ordinal>
	ordinal, ok := extractOrdinal("data-my-cluster-0-0", "my-cluster-0")
	if !ok {
		t.Fatal("expected extraction to succeed")
	}
	if ordinal != 0 {
		t.Errorf("ordinal = %d, want 0", ordinal)
	}
}

func TestExtractOrdinal_HigherOrdinal(t *testing.T) {
	ordinal, ok := extractOrdinal("data-my-cluster-0-3", "my-cluster-0")
	if !ok {
		t.Fatal("expected extraction to succeed")
	}
	if ordinal != 3 {
		t.Errorf("ordinal = %d, want 3", ordinal)
	}
}

func TestExtractOrdinal_MultiDigitOrdinal(t *testing.T) {
	ordinal, ok := extractOrdinal("data-my-cluster-0-12", "my-cluster-0")
	if !ok {
		t.Fatal("expected extraction to succeed")
	}
	if ordinal != 12 {
		t.Errorf("ordinal = %d, want 12", ordinal)
	}
}

func TestExtractOrdinal_NoMatch(t *testing.T) {
	_, ok := extractOrdinal("unrelated-pvc-name", "my-cluster-0")
	if ok {
		t.Error("should not match unrelated PVC name")
	}
}

func TestExtractOrdinal_NoTrailingDigits(t *testing.T) {
	_, ok := extractOrdinal("data-my-cluster-0-abc", "my-cluster-0")
	if ok {
		t.Error("should not match PVC without trailing digits")
	}
}

func TestExtractOrdinal_EmptyPVCName(t *testing.T) {
	_, ok := extractOrdinal("", "my-cluster-0")
	if ok {
		t.Error("should not match empty PVC name")
	}
}

func TestExtractOrdinal_OnlyDigits(t *testing.T) {
	_, ok := extractOrdinal("123", "my-cluster-0")
	if ok {
		t.Error("should not match PVC name that is only digits")
	}
}

// --- isOwnedByStatefulSet tests ---

func TestIsOwnedByStatefulSet_Matching(t *testing.T) {
	pvc := pvcWithName("data-my-sts-0")
	if !isOwnedByStatefulSet(&pvc, "my-sts") {
		t.Error("PVC should be owned by StatefulSet")
	}
}

func TestIsOwnedByStatefulSet_NotMatching(t *testing.T) {
	pvc := pvcWithName("unrelated-pvc")
	if isOwnedByStatefulSet(&pvc, "my-sts") {
		t.Error("PVC should not be owned by StatefulSet")
	}
}

func TestIsOwnedByStatefulSet_DifferentSTS(t *testing.T) {
	pvc := pvcWithName("data-other-sts-0")
	if isOwnedByStatefulSet(&pvc, "my-sts") {
		t.Error("PVC from different STS should not match")
	}
}

// --- extractOrdinal overflow ---

func TestExtractOrdinal_Overflow(t *testing.T) {
	_, ok := extractOrdinal("data-my-cluster-0-99999999999999999999", "my-cluster-0")
	if ok {
		t.Error("should not match PVC with overflowing ordinal")
	}
}

// --- extractVolumeName tests ---

func TestExtractVolumeName_Valid(t *testing.T) {
	name, ok := extractVolumeName("data-my-cluster-0-0", "my-cluster-0")
	if !ok {
		t.Fatal("expected extraction to succeed")
	}
	if name != "data" {
		t.Errorf("volume name = %q, want %q", name, "data")
	}
}

func TestExtractVolumeName_MultiPartName(t *testing.T) {
	name, ok := extractVolumeName("my-data-vol-my-cluster-0-3", "my-cluster-0")
	if !ok {
		t.Fatal("expected extraction to succeed")
	}
	if name != "my-data-vol" {
		t.Errorf("volume name = %q, want %q", name, "my-data-vol")
	}
}

func TestExtractVolumeName_Invalid(t *testing.T) {
	_, ok := extractVolumeName("unrelated-pvc-name", "my-cluster-0")
	if ok {
		t.Error("should not extract volume name from unrelated PVC")
	}
}

func TestExtractVolumeName_NoOrdinal(t *testing.T) {
	_, ok := extractVolumeName("data-my-cluster-0-", "my-cluster-0")
	if ok {
		t.Error("should not extract when ordinal is missing")
	}
}

func TestExtractVolumeName_NonNumericOrdinal(t *testing.T) {
	_, ok := extractVolumeName("data-my-cluster-0-abc", "my-cluster-0")
	if ok {
		t.Error("should not extract when ordinal is non-numeric")
	}
}
