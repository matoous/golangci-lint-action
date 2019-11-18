#!/bin/bash

set -e


OUT_FORMAT=${OUT_FORMAT:-json}
cd "$WORKING_DIRECTORY"

if [ ! -z "${INPUT_CONFIG}" ]; then CONFIG="--config $INPUT_CONFIG"; fi

if [ ! -z "${GITHUB_TOKEN}" ];
then
  sh -c "GO111MODULE=on golangci-lint run $CONFIG --out-format ${OUT_FORMAT} | golangci-lint-action"
else
  echo "Annotations inactive. No GitHub token provided"
  sh -c "GO111MODULE=on golangci-lint run $CONFIG"
fi
