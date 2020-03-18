// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package drpcmetadata define the structure of the metadata supported by drpc library.
package drpcmetadata

import "github.com/spacemonkeygo/monkit/v3"

var mon = monkit.Package()

//go:generate bash -c "go install storj.io/drpc/cmd/protoc-gen-drpc && protoc --drpc_out=plugins=drpc:. ./proto/metadata.proto"
