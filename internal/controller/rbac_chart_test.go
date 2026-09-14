package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

// The Helm chart's manager ClusterRole is hand-maintained, while
// config/rbac/role.yaml is generated from the kubebuilder markers by
// `make manifests`. Nothing regenerates the chart, so a new verb can land in one
// and not the other — which is how the missing `patch` on pods stayed invisible
// (issue #381 asks for exactly this guard). Helm is the install path most users
// take, so drift there is a production-only RBAC failure.

// readManagerRoleRules parses the generated ClusterRole.
func readManagerRoleRules(t *testing.T) []rbacv1.PolicyRule {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "config", "rbac", "role.yaml"))
	if err != nil {
		t.Fatalf("read role.yaml: %v", err)
	}

	role := &rbacv1.ClusterRole{}
	if err := yaml.Unmarshal(raw, role); err != nil {
		t.Fatalf("parse role.yaml: %v", err)
	}
	return role.Rules
}

// readChartManagerRules parses the `rules:` block of the chart template. The
// block itself is plain YAML — only the surrounding `{{- if }}` guard and the
// metadata are templated — so it is sliced out and parsed directly rather than
// shelling out to `helm template`.
func readChartManagerRules(t *testing.T) []rbacv1.PolicyRule {
	t.Helper()

	path := filepath.Join("..", "..", "charts", "aerospike-ce-kubernetes-operator",
		"templates", "clusterrole-manager.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read chart ClusterRole: %v", err)
	}

	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "rules:" {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("no rules: block in %s", path)
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "{{") {
			end = i
			break
		}
	}

	var block struct {
		Rules []rbacv1.PolicyRule `json:"rules"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[start:end], "\n")), &block); err != nil {
		t.Fatalf("parse chart rules (templating inside the rules block?): %v", err)
	}
	if len(block.Rules) == 0 {
		t.Fatalf("no rules parsed from %s", path)
	}
	return block.Rules
}

// normalizeRules renders rules order-independently so the comparison reports
// genuine differences rather than formatting ones.
func normalizeRules(rules []rbacv1.PolicyRule) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		groups := append([]string(nil), rule.APIGroups...)
		resources := append([]string(nil), rule.Resources...)
		verbs := append([]string(nil), rule.Verbs...)
		sort.Strings(groups)
		sort.Strings(resources)
		sort.Strings(verbs)
		out = append(out, fmt.Sprintf("apiGroups=%v resources=%v verbs=%v", groups, resources, verbs))
	}
	sort.Strings(out)
	return out
}

func TestChartManagerClusterRoleMatchesGeneratedRole(t *testing.T) {
	generated := normalizeRules(readManagerRoleRules(t))
	chart := normalizeRules(readChartManagerRules(t))

	if !reflect.DeepEqual(generated, chart) {
		t.Errorf("chart manager ClusterRole has drifted from config/rbac/role.yaml;"+
			" re-run `make manifests` and mirror the change in"+
			" charts/aerospike-ce-kubernetes-operator/templates/clusterrole-manager.yaml\n"+
			"generated:\n  %s\nchart:\n  %s",
			strings.Join(generated, "\n  "), strings.Join(chart, "\n  "))
	}
}

// TestManagerRoleGrantsPatchOnPods pins the specific verb the reconciler needs:
// updatePodConfigHash and adoptStorageHash patch the Pod itself, which the
// apiserver authorizes as verb=patch resource=pods (pods/status does not cover
// it). The envtest spec in rbac_test.go proves the role actually admits the
// call; this keeps the failure readable if the verb is ever dropped again.
func TestManagerRoleGrantsPatchOnPods(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rules []rbacv1.PolicyRule
	}{
		{"config/rbac/role.yaml", readManagerRoleRules(t)},
		{"chart clusterrole-manager.yaml", readChartManagerRules(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			for _, rule := range tc.rules {
				if !slices.Contains(rule.APIGroups, "") || !slices.Contains(rule.Resources, "pods") {
					continue
				}
				found = true
				if !slices.Contains(rule.Verbs, "patch") {
					t.Errorf("pods rule %v lacks the patch verb: the reconciler's pod"+
						" annotation patches would 403", rule.Verbs)
				}
			}
			if !found {
				t.Error("no rule for core pods found")
			}
		})
	}
}
