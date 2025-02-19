#!/usr/bin/env bash

set -Eeou pipefail

EXE_FILE="./dist/windows_windows_amd64_v1/binary.exe"

if [[ -f "$EXE_FILE" ]]; then
	echo "signing Windows binary: ${EXE_FILE}"

  docker run \
    -e GRS_CONFIG_USER1_USERNAME="${ARTIFACTORY_SIGN_USER}" \
    -e GRS_CONFIG_USER1_PASSWORD="${ARTIFACTORY_SIGN_PASSWORD}" \
		--rm -v "$(pwd)":"$(pwd)" -w "$(pwd)" \
    "${ARTIFACTORY_REGISTRY}/release-tools-container-registry-local/garasign-jsign" \
		/bin/bash -c "jsign --tsaurl http://timestamp.digicert.com -a ${AUTHENTICODE_KEY_NAME} \"${EXE_FILE}\""
fi
