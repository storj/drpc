#!/usr/bin/env bash

set -e

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

PATH_COND=(-path)
if [ "$1" == "-v" ]; then
	PATH_COND=(! -path)
	shift
fi

GLOB="$1"
shift

find "${SCRIPTDIR}/.." "${PATH_COND[@]}" '*'"${GLOB}"'*' -name "go.mod" -exec dirname {} + \
| while read -r DIR; do
	(cd "${DIR}" || exit; "$@")
done
