// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package backcompat

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/errs"

	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcsignal"
)

func TestCompatibility(t *testing.T) {
	for _, client := range []string{"old", "new"} {
		for _, server := range []string{"old", "new"} {
			client, server := client, server
			t.Run(fmt.Sprintf("%s_client_%s_server", client, server), func(t *testing.T) {
				testCombination(t,
					fmt.Sprintf("./%sservice", client),
					fmt.Sprintf("./%sservice", server))
			})
		}
	}
}

func testCombination(t *testing.T, client, server string) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	sig := new(drpcsignal.Signal)
	addrCh := make(chan string, 1)

	// launch the server
	ctx.Run(func(ctx context.Context) {
		err := runTestServer(ctx, server, addrCh)
		if err != nil {
			sig.Set(err)
		}
	})

	// launch the client
	ctx.Run(func(ctx context.Context) {
		err := runTestClient(ctx, client, addrCh)
		if err != nil {
			sig.Set(err)
		}
	})

	// launch a goroutine to set the signal if the above goroutines exit.
	go func() {
		ctx.Wait()
		sig.Set(nil)
	}()

	// wait for the signal to be set for any reason and assert no error.
	<-sig.Signal()
	assert.NoError(t, sig.Err())
}

func runTestServer(ctx context.Context, server string, addrCh chan string) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer close(addrCh)

	var stderr bytes.Buffer
	defer func() {
		if err != nil {
			fmt.Println(&stderr)
		}
	}()

	cmd := exec.Command("go", "run", ".", "server", ":0")
	cmd.Stderr = &stderr
	cmd.Dir = server

	rc, err := cmd.StdoutPipe()
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = rc.Close() }()

	if err := cmd.Start(); err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()

	addr, err := bufio.NewReader(rc).ReadString('\n')
	if err != nil {
		return errs.Wrap(err)
	}
	addrCh <- strings.TrimSpace(addr)

	return cmd.Wait()
}

func runTestClient(ctx context.Context, client string, addrCh chan string) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var addr string
	var ok bool

	select {
	case <-ctx.Done():
	case addr, ok = <-addrCh:
	}
	if !ok {
		return nil
	}

	var stderr bytes.Buffer
	defer func() {
		if err != nil {
			fmt.Println(&stderr)
		}
	}()

	cmd := exec.Command("go", "run", ".", "client", addr) //nolint:gosec
	cmd.Stderr = &stderr
	cmd.Dir = client

	if err := cmd.Start(); err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()

	return cmd.Wait()
}
