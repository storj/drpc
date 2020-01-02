#!/usr/bin/env bash

set -e

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(cd "${SCRIPTDIR}/..";                     go mod tidy)
(cd "${SCRIPTDIR}/../internal/grpccompat"; go mod tidy)
