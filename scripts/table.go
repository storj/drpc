// +build ignore

// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var kindMap = map[string]string{
	"Unitary":             "Unitary",
	"InputStream":         "Input Stream",
	"OutputStream":        "Output Stream",
	"BidirectionalStream": "Bidir Stream",
}

var unitMap = map[string]string{
	"time/op":   "time/op",
	"speed":     "speed",
	"alloc/op":  "mem/op",
	"allocs/op": "allocs/op",
}

func popPercent(x []string) []string {
	for !strings.Contains(x[0], "%") {
		x = x[1:]
	}
	return x[1:]
}

func tryInt(x string) string {
	if n, err := strconv.ParseFloat(x, 64); err == nil && float64(int64(n)) == n {
		return fmt.Sprint(int64(n))
	}
	return x
}

type key struct {
	lib  string // DRPC vs gRPC vs delta
	kind string // unitary|input|output|bidir
	size string // small|med|large
	unit string // time/speed/alloc/mem
}

func (k key) withLib(lib string) key   { k.lib = lib; return k }
func (k key) withKind(kind string) key { k.kind = kind; return k }
func (k key) withSize(size string) key { k.size = size; return k }
func (k key) withUnit(unit string) key { k.unit = unit; return k }

func main() {
	data := make(map[key]string)

	var unit string

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		} else if fields[0] == "name" {
			unit = fields[2]
			continue
		}
		parts := strings.SplitN(fields[0], "/", 2)

		k := key{
			kind: kindMap[parts[0]],
			size: strings.SplitN(parts[1], "-", 2)[0],
			unit: unitMap[unit],
		}

		fields = fields[1:]
		data[k.withLib("grpc")] = tryInt(fields[0])
		fields = popPercent(fields)
		data[k.withLib("drpc")] = tryInt(fields[0])
		fields = popPercent(fields)
		data[k.withLib("delta")] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Printf("<table>\n")
	fmt.Printf("    <tr>\n")
	fmt.Printf("        <td rowspan=2>Measure</td>\n")
	fmt.Printf("        <td rowspan=2>Benchmark</td><td rowspan=2></td>\n")
	fmt.Printf("        <td colspan=3>Small</td><td rowspan=2></td>\n")
	fmt.Printf("        <td colspan=3>Medium</td><td rowspan=2></td>\n")
	fmt.Printf("        <td colspan=3>Large</td>\n")
	fmt.Printf("    </tr>\n")
	fmt.Printf("    <tr>\n")
	fmt.Printf("        <td>gRPC</td><td>DRPC</td><td>delta</td>\n")
	fmt.Printf("        <td>gRPC</td><td>DRPC</td><td>delta</td>\n")
	fmt.Printf("        <td>gRPC</td><td>DRPC</td><td>delta</td>\n")
	fmt.Printf("    </tr>\n")

	iterLibs := func(k key, next func(n int, k key)) {
		next(0, k.withLib("grpc"))
		next(1, k.withLib("drpc"))
		next(2, k.withLib("delta"))
	}

	iterKinds := func(k key, next func(n int, k key)) {
		next(0, k.withKind("Unitary"))
		next(1, k.withKind("Input Stream"))
		next(2, k.withKind("Output Stream"))
		next(3, k.withKind("Bidir Stream"))
	}

	iterSizes := func(k key, next func(n int, k key)) {
		next(0, k.withSize("Small"))
		next(1, k.withSize("Med"))
		next(2, k.withSize("Large"))
	}

	iterUnits := func(k key, next func(n int, k key)) {
		next(0, k.withUnit("time/op"))
		next(1, k.withUnit("speed"))
		next(2, k.withUnit("mem/op"))
		next(3, k.withUnit("allocs/op"))
	}

	iterUnits(key{}, func(_ int, k key) {
		fmt.Printf("    <tr><td colspan=14></td></tr>\n")
		iterKinds(k, func(n int, k key) {
			fmt.Printf("    <tr>\n")
			if n == 0 {
				fmt.Printf("        <td rowspan=4>%s</td>\n", k.unit)
			}
			fmt.Printf("        <td>%s</td>", k.kind)
			if n == 0 {
				fmt.Printf("<td rowspan=4></td>\n")
			} else {
				fmt.Printf("\n")
			}
			iterSizes(k, func(m int, k key) {
				fmt.Printf("        ")
				iterLibs(k, func(_ int, k key) {
					fmt.Printf("<td>%s</td>", data[k])
				})
				if n == 0 && m < 2 {
					fmt.Printf("<td rowspan=4></td>\n")
				} else {
					fmt.Printf("\n")
				}
			})
			fmt.Printf("    </tr>\n")
		})
	})
	fmt.Printf("</table>\n")
}
