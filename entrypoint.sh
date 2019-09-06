#!/bin/bash

set -e

cd "$GITHUB_WORKSPACE"

if [ ! -z "${INPUT_CONFIG}" ]; then CONFIG="--config $INPUT_CONFIG"; fi

if [ ! -z "${GITHUB_TOKEN}" ];
then
  sh -c "golangci-lint run $CONFIG --out-format json | golangci-lint-action"
else
  echo "Annotations inactive. No GitHub token provided"
  sh -c "golangci-lint run $CONFIG"
fi
