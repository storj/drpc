#!/usr/bin/env bash

set -e

log() {
	echo "---" "$@"
}

# put the godocdown template in a temporary directory
TEMPLATE=$(mktemp)
trap 'rm ${TEMPLATE}' EXIT

cat <<EOF >"${TEMPLATE}"
# package {{ .Name }}

\`import "{{ .ImportPath }}"\`

{{ .EmitSynopsis }}

{{ .EmitUsage }}
EOF

# build the godocdown tool
GODOCDOWN=$(
	SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
	cd "${SCRIPTDIR}"
	cd "$(pwd -P)"

	IMPORT=github.com/robertkrimen/godocdown/godocdown
	go install -v "${IMPORT}"
	go list -f '{{ .Target }}' "${IMPORT}"
)

# walk all the packages and generate docs
PACKAGES=$(go list -f '{{ .ImportPath }}|{{ .Dir }}' storj.io/drpc/...)
for DESC in ${PACKAGES}; do
	PACKAGE="$(echo "${DESC}" | cut -d '|' -f 1)"
	DIR="$(echo "${DESC}" | cut -d '|' -f 2)"

	log "generating docs for ${PACKAGE}..."
	"${GODOCDOWN}" -template "${TEMPLATE}" "${DIR}" > "${DIR}/README.md"
done
