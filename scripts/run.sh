#!/usr/bin/env bash

set -e

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(cd "${SCRIPTDIR}/.." || exit;                     "$@")
(cd "${SCRIPTDIR}/../internal/grpccompat" || exit; "$@")
