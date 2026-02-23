#!/bin/bash
set -e

# Configuration directory
CONFIG_SRC="/configmap"
CONFIG_DST="/etc/aerospike"

echo "Aerospike CE Operator Init Container"
echo "======================================"

# Copy configuration files
echo "Copying aerospike.conf from ConfigMap..."
if [ -f "${CONFIG_SRC}/aerospike.conf" ]; then
    cp "${CONFIG_SRC}/aerospike.conf" "${CONFIG_DST}/aerospike.conf"
    echo "Configuration copied successfully."
else
    echo "ERROR: aerospike.conf not found in ConfigMap!"
    exit 1
fi

# Get pod information from environment
POD_NAME=${POD_NAME:-$(hostname)}
POD_IP=${POD_IP:-$(hostname -i 2>/dev/null || echo "127.0.0.1")}
NODE_IP=${NODE_IP:-""}

echo "Pod Name: ${POD_NAME}"
echo "Pod IP: ${POD_IP}"
echo "Node IP: ${NODE_IP}"

# Replace placeholders in config
if [ -n "${POD_IP}" ]; then
    sed -i "s/MY_POD_IP/${POD_IP}/g" "${CONFIG_DST}/aerospike.conf"
fi
if [ -n "${POD_NAME}" ]; then
    sed -i "s/MY_POD_NAME/${POD_NAME}/g" "${CONFIG_DST}/aerospike.conf"
fi
if [ -n "${NODE_IP}" ]; then
    sed -i "s/MY_NODE_IP/${NODE_IP}/g" "${CONFIG_DST}/aerospike.conf"
fi

# Helper: process volume operations (used by both WIPE and INIT)
process_volumes() {
    local label="$1"
    local volumes="$2"

    if [ -z "${volumes}" ]; then
        return
    fi

    echo "Processing ${label}..."
    IFS=',' read -ra VOLS <<< "${volumes}"
    for vol_spec in "${VOLS[@]}"; do
        IFS=':' read -ra PARTS <<< "${vol_spec}"
        method="${PARTS[0]}"
        path="${PARTS[1]}"

        case "${method}" in
            deleteFiles)
                echo "[${label}] Deleting files in ${path}..."
                rm -rf "${path:?}"/*
                ;;
            dd)
                echo "[${label}] Zeroing first 1MB of ${path}..."
                dd if=/dev/zero of="${path}" bs=1M count=1 conv=notrunc 2>/dev/null
                ;;
            blkdiscard)
                echo "[${label}] Discarding blocks on ${path}..."
                blkdiscard "${path}" 2>/dev/null || echo "blkdiscard failed for ${path}, continuing..."
                ;;
            headerCleanup)
                echo "[${label}] Cleaning Aerospike headers on ${path}..."
                dd if=/dev/zero of="${path}" bs=4096 count=1 conv=notrunc 2>/dev/null
                ;;
            blkdiscardWithHeaderCleanup)
                echo "[${label}] Discarding blocks and cleaning headers on ${path}..."
                blkdiscard "${path}" 2>/dev/null || echo "blkdiscard failed for ${path}, continuing..."
                dd if=/dev/zero of="${path}" bs=4096 count=1 conv=notrunc 2>/dev/null
                ;;
            *)
                echo "[${label}] Skipping ${path} (method: ${method})"
                ;;
        esac
    done
}

# Wipe dirty volumes (runs before init, only for volumes marked dirty)
WIPE_VOLUMES="${WIPE_VOLUMES:-}"
process_volumes "WIPE" "${WIPE_VOLUMES}"

# Volume initialization
INIT_VOLUMES="${INIT_VOLUMES:-}"
process_volumes "INIT" "${INIT_VOLUMES}"

echo "Init container completed successfully."
