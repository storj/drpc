// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

// +build tools

package tools

import (
	_ "github.com/robertkrimen/godocdown/godocdown"

	_ "github.com/storj/ci/check-atomic-align"
	_ "github.com/storj/ci/check-copyright"
	_ "github.com/storj/ci/check-errs"
	_ "github.com/storj/ci/check-imports"
	_ "github.com/storj/ci/check-large-files"
	_ "github.com/storj/ci/check-monkit"
	_ "github.com/storj/ci/check-peer-constraints"
)
