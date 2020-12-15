#!/usr/bin/env bash

set -e

cd "$( dirname "${BASH_SOURCE[0]}" )"

(
    cat tools.go \
        | grep _              \
        | awk '{print $2}'    \
        | cut -d'"' -f2
) | while read -r CMD; do
    echo "--- installing" "$CMD"
    go install "$CMD"
done
