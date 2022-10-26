// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeebo/assert"

	"storj.io/drpc/drpchttp"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpctest"
)

func TestHTTP(t *testing.T) {
	ctx := drpctest.NewTracker(t)
	defer ctx.Close()

	mux := drpcmux.New()
	assert.NoError(t, DRPCRegisterService(mux, standardImpl))

	server := httptest.NewServer(drpchttp.New(mux))
	defer server.Close()

	type response struct {
		StatusCode int
		Code       string
		Msg        string
		Response   *jsonOut
	}

	request := func(method, body string, metadata ...string) (r response) {
		req, err := http.NewRequestWithContext(ctx, "POST", server.URL+method, strings.NewReader(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header["X-Drpc-Metadata"] = metadata

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)

		data, err := io.ReadAll(resp.Body)
		assert.NoError(t, resp.Body.Close())
		assert.NoError(t, err)

		if resp.StatusCode == http.StatusOK {
			assert.NoError(t, json.Unmarshal(data, &r.Response))
		} else {
			assert.NoError(t, json.Unmarshal(data, &r))
		}
		r.StatusCode = resp.StatusCode

		return r
	}

	assertEqual := func(t *testing.T, a, b response) {
		t.Helper()
		assert.True(t, Equal((*Out)(a.Response), (*Out)(b.Response)))
		a.Response, b.Response = nil, nil
		assert.DeepEqual(t, a, b)
	}

	// basic successful request
	assertEqual(t, request("/service.Service/Method1", `{"in": 1}`), response{
		StatusCode: http.StatusOK,
		Response:   &jsonOut{Out: 1},
	})

	// basic erroring request
	assertEqual(t, request("/service.Service/Method1", `{"in": 5}`), response{
		StatusCode: http.StatusInternalServerError,
		Code:       "drpcerr(5)",
		Msg:        "test",
	})

	// metadata gets passed through
	assertEqual(t, request("/service.Service/Method1", `{"in": 1}`, "inc=10"), response{
		StatusCode: http.StatusOK,
		Response:   &jsonOut{Out: 11},
	})

	// non-existing method
	assertEqual(t, request("/service.Service/DoesNotExist", `{}`), response{
		StatusCode: http.StatusInternalServerError,
		Code:       "unknown",
		Msg:        `protocol error: unknown rpc: "/service.Service/DoesNotExist"`,
	})
}

//
// super hacky hack to make it so that you can use encoding/json with the protobuf
//

type jsonOut Out

func (o *jsonOut) UnmarshalJSON(v []byte) error {
	return Encoding.JSONUnmarshal(v, (*Out)(o))
}
