// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeebo/errs"

	"storj.io/drpc/drpcmetadata"
)

//
// code to unescape and build the request context metadata
//

// Context returns the context.Context from the http.Request with any metadata
// sent using the X-Drpc-Metadata header set as values.
func Context(req *http.Request) (context.Context, error) {
	// header string we look up must already be canonicalized
	return buildContext(req.Context(), req.Header["X-Drpc-Metadata"])
}

// buildContext adds key/value pairs in entries that are of the form
// `urlencode(key)=urlencode(value)` to the passed in context.
func buildContext(ctx context.Context, entries []string) (context.Context, error) {
	for _, entry := range entries {
		var key, value string
		var err error

		index := strings.IndexByte(entry, '=')
		if index >= 0 {
			value, err = unescape(entry[index+1:])
			if err != nil {
				return nil, err
			}
			entry = entry[:index]
		}

		key, err = unescape(entry)
		if err != nil {
			return nil, err
		}

		ctx = drpcmetadata.Add(ctx, key, value)
	}

	return ctx, nil
}

// unhex adds to the accumulator c the numeric value of the hex digit v
// multiplied by the multiplier m and a boolean indicating if the hex
// digit was valid. it compiles to like 3 compares and can be inlined.
func unhex(c, v, m byte) (d byte, ok bool) {
	switch {
	case '0' <= v && v <= '9':
		d = (v - '0')
	case 'a' <= v && v <= 'f':
		d = (v - 'a' + 10)
	case 'A' <= v && v <= 'F':
		d = (v - 'A' + 10)
	default:
		return 0, false
	}
	return c + d*m, true
}

// unescape is an optimized form of url.QueryUnescape that is less general.
func unescape(s string) (string, error) {
	count := strings.Count(s, "%")
	if count == 0 {
		return s, nil
	}

	var t strings.Builder
	t.Grow(len(s) - 2*count)

	for i := uint(0); i < uint(len(s)); i++ {
		switch s[i] {
		case '%':
			if i+2 >= uint(len(s)) {
				return "", errs.New("error unescaping %q: sequence ends", s)
			}

			c, ok := unhex(0, s[i+1], 16)
			if !ok {
				return "", errs.New("error unescaping %q: invalid hex digit", s)
			}

			c, ok = unhex(c, s[i+2], 1)
			if !ok {
				return "", errs.New("error unescaping %q: invalid hex digit", s)
			}

			_ = t.WriteByte(c)
			i += 2

		default:
			_ = t.WriteByte(s[i])
		}
	}

	return t.String(), nil
}
