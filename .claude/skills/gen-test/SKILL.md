---
name: gen-test
description: Generate Ginkgo v2 or standard Go tests following project conventions
disable-model-invocation: false
---

# Generate Test

Generate tests for the specified Go package following this project's established patterns.

## Arguments

The user provides a package path as argument (e.g., `/gen-test internal/storage`).

## Test Style Decision

Choose the test style based on what's being tested:

### Standard Go Tests (default for most packages)
Use for: `internal/configgen/`, `internal/storage/`, `internal/podutil/`, `internal/utils/`, `api/v1alpha1/` (webhook tests)

Conventions:
- Standard `func Test*(t *testing.T)` signatures
- Naming: `Test<FunctionName>_<Scenario>` (e.g., `TestBuildInitContainer_UsesClusterImage`)
- One test per scenario, not subtests
- Use `t.Fatalf()` for fatal setup errors, `t.Errorf()` for assertion failures
- Use `t.Helper()` in assertion helper functions
- Table-driven tests with `[]struct{ ... }` for parameterized cases
- Factory functions like `newTestCluster()` for repeated setup
- Helper `boolPtr(b bool) *bool` for pointer fields
- Map conversions for env var / label assertions (convert slices to maps)

Example pattern:
```go
func TestFunctionName_Scenario(t *testing.T) {
    // Setup
    cluster := &v1alpha1.AerospikeCECluster{...}

    // Act
    result := FunctionUnderTest(cluster)

    // Assert
    if result != expected {
        t.Errorf("FunctionUnderTest() = %v, want %v", result, expected)
    }
}
```

### Ginkgo v2 + envtest (controller integration tests)
Use for: `internal/controller/` when testing reconciliation against a real API server

Conventions:
- Dot imports: `. "github.com/onsi/ginkgo/v2"` and `. "github.com/onsi/gomega"`
- Suite file with `BeforeSuite`/`AfterSuite` for envtest setup
- `Describe`/`Context`/`It` BDD structure
- `Eventually()` for async assertions with configurable timeout/polling
- `By()` for narrative steps
- `Expect(x).To(Equal(y))` assertion style

### Ginkgo v2 BDD (e2e tests)
Use for: `test/e2e/` only

Conventions:
- Build tag: `//go:build e2e`
- `Ordered` option for sequential execution
- `BeforeAll`/`AfterAll` for one-time setup
- `AfterEach` with `CurrentSpecReport()` for failure diagnostics
- `Eventually(func(g Gomega) { ... }).Should(Succeed())` pattern

## Test Coverage Priorities

When generating tests, prioritize:
1. Happy path (normal operation)
2. Edge cases (nil inputs, empty slices, zero values)
3. Error conditions (invalid inputs, constraint violations)
4. Preservation tests (ensure existing values aren't overwritten by defaults)

## Important Notes

- Read the target package source code FIRST before generating tests
- Read existing tests in the same package to match local style
- Use existing types from `api/v1alpha1/` for test data construction
- For Kubernetes objects, import from `k8s.io/api/core/v1`, `k8s.io/apimachinery/pkg/apis/meta/v1`
- Run `go test ./<package-path>/ -v` after generating to verify tests pass
- Run `make lint` to ensure no lint violations
