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

# walk all the packages and generate docs
PACKAGES=$(go list -f '{{ .ImportPath }}|{{ .Dir }}' storj.io/drpc/...)
for DESC in ${PACKAGES}; do
	PACKAGE="$(echo "${DESC}" | cut -d '|' -f 1)"
	DIR="$(echo "${DESC}" | cut -d '|' -f 2)"

	if [[ "$PACKAGE" != "storj.io/drpc" ]]; then
		log "generating docs for ${PACKAGE}..."
		godocdown -template "${TEMPLATE}" "${DIR}" > "${DIR}/README.md"
	fi
done
