// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	const baseURL = "http://localhost:8080"

	// make the http request
	resp, err := http.Post(baseURL+"/sesamestreet.CookieMonster/EatCookie",
		"application/json", strings.NewReader(`{"type": "Chocolate"}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// confirm the http layer worked okay
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http status %q", resp.StatusCode)
	}

	// parse the response
	var data struct {
		Status string `json:"status"`

		Response struct {
			Cookie struct {
				Type string `json:"type"`
			} `json:"cookie"`
		} `json:"response"`

		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return err
	}

	// confirm the rpc layer worked okay
	if data.Status != "ok" {
		return fmt.Errorf("unexpected rpc status %q %q %q",
			data.Status, data.Code, data.Error)
	}

	// check the results
	_, err = fmt.Println(data.Response.Cookie.Type)
	return err
}
