#!/usr/bin/env bash

set -e

(
	echo '| Package | Lines |'
	echo '| --- | --- |'
	go list -f '{{.ImportPath}} {{.Dir}}' storj.io/drpc/... \
		| while read -r -a line; do
			echo -n "| ${line[0]} | "
			find "${line[1]}" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -print0 \
				| xargs -0 cloc --json --quiet 2>/dev/null \
				| jq -jr '.Go.code'
			echo " |"
		done \
	| sort -rnk4 \
	| awk '{total+=$4; print $0} END {print "| **Total** | **" total "** |"}'
) | column -to ' '