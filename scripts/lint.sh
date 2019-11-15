#!/usr/bin/env bash

set -e

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(cd "${SCRIPTDIR}/..";                     staticcheck ./... && golangci-lint -j=2 run)
(cd "${SCRIPTDIR}/../internal/grpccompat"; staticcheck ./... && golangci-lint -j=2 run)
