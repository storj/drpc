// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

// newservice runs the new version of the backwards compatibility check.
package main

import (
	"context"

	"storj.io/drpc/internal/backcompat"
)

func main() { backcompat.Main(context.Background()) }
