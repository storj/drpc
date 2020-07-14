// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Package drpcstream sends protobufs using the dprc wire protocol.
//
// ![Stream state machine diagram](state.png)
package drpcstream

//go:generate bash -c "dot -Tpng -o state.png state.dot"

import "github.com/spacemonkeygo/monkit/v3"

var mon = monkit.Package()
