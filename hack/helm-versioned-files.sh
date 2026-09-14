#!/usr/bin/env bash
# helm-versioned-files.sh: the single list of paths that carry the in-repo Helm
# chart version. Sourced, never executed.
#
# Three consumers have to agree on this list, and when they drifted apart the
# chart pin in the translated install guide was rewritten by helm-set-version.sh
# but never staged by the release workflow, so `make helm-check-version` went red
# on main after every auto-bump:
#
#   * hack/helm-set-version.sh    rewrites these paths
#   * hack/helm-check-version.sh  asserts the pins in VERSIONED_DOCS are in sync
#   * .github/workflows/daily-release.yml stages VERSIONED_PATHS after a bump,
#     via `hack/helm-set-version.sh --print-files`
#
# Callers must set REPO_ROOT before sourcing. Paths are repo-root-relative so the
# workflow can hand them straight to `git add`.

# Docs that document a chart version pin (`--version`, `targetRevision:`,
# `version:`). `helm install oci://... --version 1.3.1` installs a stale chart
# just as surely as a stale in-repo Chart.yaml does.
VERSIONED_DOCS=(
  "charts/aerospike-ce-kubernetes-operator/README.md"
  "docs/i18n/ko/docusaurus-plugin-content-docs/current/getting-started/install.md"
)

# Everything helm-set-version.sh rewrites, docs included. The vendored sub-chart
# directory is listed rather than the .tgz inside it, because the archive's name
# carries the version and so changes on every bump.
# shellcheck disable=SC2034  # consumed by the scripts that source this file.
VERSIONED_PATHS=(
  "charts/aerospike-ce-kubernetes-operator-crds/Chart.yaml"
  "charts/aerospike-ce-kubernetes-operator/Chart.yaml"
  "charts/aerospike-ce-kubernetes-operator/Chart.lock"
  "charts/aerospike-ce-kubernetes-operator/charts"
  "${VERSIONED_DOCS[@]}"
)
