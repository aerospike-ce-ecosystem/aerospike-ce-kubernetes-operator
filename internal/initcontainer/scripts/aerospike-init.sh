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

# Volume initialization
INIT_VOLUMES="${INIT_VOLUMES:-}"
if [ -n "${INIT_VOLUMES}" ]; then
    echo "Processing volume initialization..."
    IFS=',' read -ra VOLS <<< "${INIT_VOLUMES}"
    for vol_spec in "${VOLS[@]}"; do
        IFS=':' read -ra PARTS <<< "${vol_spec}"
        method="${PARTS[0]}"
        path="${PARTS[1]}"

        case "${method}" in
            deleteFiles)
                echo "Deleting files in ${path}..."
                rm -rf "${path:?}"/*
                ;;
            dd)
                echo "Zeroing first 1MB of ${path}..."
                dd if=/dev/zero of="${path}" bs=1M count=1 conv=notrunc 2>/dev/null
                ;;
            blkdiscard)
                echo "Discarding blocks on ${path}..."
                blkdiscard "${path}" 2>/dev/null || echo "blkdiscard failed for ${path}, continuing..."
                ;;
            headerCleanup)
                echo "Cleaning Aerospike headers on ${path}..."
                dd if=/dev/zero of="${path}" bs=4096 count=1 conv=notrunc 2>/dev/null
                ;;
            *)
                echo "Skipping initialization for ${path} (method: ${method})"
                ;;
        esac
    done
fi

echo "Init container completed successfully."
