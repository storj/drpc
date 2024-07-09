// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"bytes"
	"errors"
	"io"
	"runtime/trace"
	"sort"
	"testing"

	"github.com/zeebo/assert"
	exptrace "golang.org/x/exp/trace"

	"storj.io/drpc/drpctest"
)

func TestRuntimeTracing(t *testing.T) {
	if trace.IsEnabled() {
		t.Skip("tracing already enabled")
	}

	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	cli, close := createConnection(t, standardImpl)
	defer close()

	var buf bytes.Buffer
	assert.NoError(t, trace.Start(&buf))

	out, err := cli.Method1(ctx, &In{In: 1})
	close()

	trace.Stop()

	assert.NoError(t, err)
	assert.True(t, Equal(out, &Out{Out: 1}))

	r, err := exptrace.NewReader(&buf)
	assert.NoError(t, err)

	var events []string
	for {
		ev, err := r.ReadEvent()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)

		switch ev.Kind() {
		case exptrace.EventTaskBegin:
			events = append(events, "begin "+ev.Task().Type)
		case exptrace.EventTaskEnd:
			events = append(events, "end "+ev.Task().Type)
		}
	}

	sort.Strings(events) // srv and client end events can be in any order

	assert.Equal(t, events, []string{
		"begin cli/service.Service/Method1",
		"begin srv/service.Service/Method1",
		"end cli/service.Service/Method1",
		"end srv/service.Service/Method1",
	})
}
