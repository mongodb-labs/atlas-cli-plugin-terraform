#!/usr/bin/env bash

set -Eeou pipefail

if [[ -f "${artifact:?}" ]]; then
  echo "notarizing package ${artifact}"

  docker run \
    -e GRS_CONFIG_USER1_USERNAME="${ARTIFACTORY_SIGN_USER}" \
    -e GRS_CONFIG_USER1_PASSWORD="${ARTIFACTORY_SIGN_PASSWORD}" \
    --rm -v "$(pwd)":"$(pwd)" -w "$(pwd)" \
    "${ARTIFACTORY_REGISTRY}/release-tools-container-registry-local/garasign-gpg" \
    /bin/bash -c "gpgloader && gpg --yes -v --armor -o ${artifact}.sig --detach-sign ${artifact}"
fi

echo "Signing of ${artifact} completed."
