package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseImageVersion extracts the major, minor, and patch version from a
// container image string such as "aerospike:ce-8.1.1.1".
func ParseImageVersion(image string) (major, minor, patch int, err error) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return 0, 0, 0, fmt.Errorf("image %q has no tag", image)
	}

	tag := parts[1]

	// Strip known tag prefixes (e.g., "ce-7.2.0" → "7.2.0").
	for _, prefix := range []string{"ce-", "ee-"} {
		if after, found := strings.CutPrefix(tag, prefix); found {
			tag = after

			break
		}
	}

	versionParts := strings.SplitN(tag, ".", 3)
	if len(versionParts) != 3 {
		return 0, 0, 0, fmt.Errorf("tag %q does not follow major.minor.patch format", tag)
	}

	major, err = strconv.Atoi(versionParts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version %q: %w", versionParts[0], err)
	}

	minor, err = strconv.Atoi(versionParts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version %q: %w", versionParts[1], err)
	}

	// Strip any suffix after the patch number (e.g., "0-rc1", "0.6_1").
	patchStr := versionParts[2]
	if idx := strings.IndexAny(patchStr, "-+._"); idx != -1 {
		patchStr = patchStr[:idx]
	}

	patch, err = strconv.Atoi(patchStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version %q: %w", versionParts[2], err)
	}

	return major, minor, patch, nil
}

// IsEnterpriseImage returns true if the image is an Enterprise Edition image.
// Detects both legacy format (containing "enterprise") and new official format (":ee-" tag prefix).
func IsEnterpriseImage(image string) bool {
	if strings.Contains(strings.ToLower(image), "enterprise") {
		return true
	}

	parts := strings.SplitN(image, ":", 2)
	if len(parts) == 2 && strings.HasPrefix(strings.ToLower(parts[1]), "ee-") {
		return true
	}

	return false
}
