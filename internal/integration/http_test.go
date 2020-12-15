// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/zeebo/assert"

	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcmux"
)

func TestHTTP(t *testing.T) {
	ctx := drpcctx.NewTracker(context.Background())
	defer ctx.Wait()
	defer ctx.Cancel()

	mux := drpcmux.New()
	assert.NoError(t, DRPCRegisterService(mux, standardImpl))

	server := httptest.NewServer(mux)
	defer server.Close()

	type response struct {
		Status   string
		Code     int
		Error    string
		Response *jsonOut
	}

	request := func(method, body string, metadata ...string) (r response) {
		req, err := http.NewRequest("POST", server.URL+method, strings.NewReader(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header["X-Drpc-Metadata"] = metadata

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)

		data, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, resp.Body.Close())
		assert.NoError(t, err)

		assert.NoError(t, json.Unmarshal(data, &r))
		return r
	}

	// basic successful request
	assert.DeepEqual(t, request("/service.Service/Method1", `{"in": 1}`), response{
		Status:   "ok",
		Response: &jsonOut{Out: 1},
	})

	// basic erroring request
	assert.DeepEqual(t, request("/service.Service/Method1", `{"in": 5}`), response{
		Status: "error",
		Error:  "test",
		Code:   5,
	})

	// metadata gets passed through
	assert.DeepEqual(t, request("/service.Service/Method1", `{"in": 1}`, "inc=10"), response{
		Status:   "ok",
		Response: &jsonOut{Out: 11},
	})

	// non-existing method
	assert.DeepEqual(t, request("/service.Service/DoesNotExist", `{}`), response{
		Status: "error",
		Error:  `protocol error: unknown rpc: "/service.Service/DoesNotExist"`,
	})

	// non-unitary methods
	assert.DeepEqual(t, request("/service.Service/Method2", `{}`), response{
		Status: "error",
		Error:  `protocol error: non-unitary rpc: "/service.Service/Method2"`,
	})
	assert.DeepEqual(t, request("/service.Service/Method3", `{}`), response{
		Status: "error",
		Error:  `protocol error: non-unitary rpc: "/service.Service/Method3"`,
	})
	assert.DeepEqual(t, request("/service.Service/Method4", `{}`), response{
		Status: "error",
		Error:  `protocol error: non-unitary rpc: "/service.Service/Method4"`,
	})
}

//
// super hacky hack to make it so that you can use encoding/json with the protobuf
//

type jsonOut Out

func (o *jsonOut) UnmarshalJSON(v []byte) error {
	return jsonpb.Unmarshal(bytes.NewReader(v), (*Out)(o))
}
