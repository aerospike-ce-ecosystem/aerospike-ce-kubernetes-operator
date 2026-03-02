#!/usr/bin/env bash
# helm-sync-crds.sh: Sync generated CRDs into acko-crds chart templates/,
# injecting the helm.sh/resource-policy: keep annotation to prevent accidental
# deletion on `helm uninstall`.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CRD_SRC_DIR="${REPO_ROOT}/config/crd/bases"
CRD_DST_DIR="${REPO_ROOT}/charts/aerospike-ce-kubernetes-operator-crds/templates"

sync_crd() {
  local src="${CRD_SRC_DIR}/$1"
  local dst="${CRD_DST_DIR}/$2"

  if [[ ! -f "${src}" ]]; then
    echo "ERROR: source CRD not found: ${src}" >&2
    exit 1
  fi

  # Copy source and inject helm.sh/resource-policy: keep after the existing
  # controller-gen annotation line (which is always present in generated CRDs).
  # Uses awk for portability across macOS and Linux.
  awk '
    /controller-gen\.kubebuilder\.io\/version/ {
      print
      print "    \"helm.sh/resource-policy\": keep"
      next
    }
    { print }
  ' "${src}" > "${dst}"

  echo "Synced: $1 → charts/aerospike-ce-kubernetes-operator-crds/templates/$2"
}

sync_crd "acko.io_aerospikeclusters.yaml"        "aerospikecluster-crd.yaml"
sync_crd "acko.io_aerospikeclustertemplates.yaml" "aerospikeclustertemplate-crd.yaml"

echo "CRD sync complete."
