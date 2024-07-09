# [![DRPC](logo.png)](https://storj.github.io/drpc/)

A drop-in, lightweight gRPC replacement.

[![Go Report Card](https://goreportcard.com/badge/storj.io/drpc)](https://goreportcard.com/report/storj.io/drpc)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/storj.io/drpc)
![Beta](https://img.shields.io/badge/version-beta-green.svg)
[![Zulip Chat](https://img.shields.io/badge/zulip-join_chat-brightgreen.svg)](https://drpc.zulipchat.com)

## Links

 * [DRPC website](https://storj.github.io/drpc/)
 * [Examples](https://github.com/storj/drpc/tree/main/examples)
 * [Quickstart documentation](https://storj.github.io/drpc/docs.html)
 * [Launch blog post](https://www.storj.io/blog/introducing-drpc-our-replacement-for-grpc)

## Highlights

* Simple, at just a few thousand [lines of code](#lines-of-code).
* [Small dependencies](./blob/main/go.mod). Only 3 requirements in go.mod, and 9 lines of `go mod graph`!
* Compatible. Works for many gRPC use-cases as-is!
* [Fast](#benchmarks). DRPC has a lightning quick [wire format](https://github.com/storj/drpc/wiki/Docs:-Wire-protocol).
* [Extensible](#external-packages). DRPC is transport agnostic, supports middleware, and is designed around interfaces.
* Battle Tested. Already used in production for years across tens of thousands of servers.

## External Packages

 * [go.bryk.io/pkg/net/drpc](https://pkg.go.dev/go.bryk.io/pkg/net/drpc)
    - Simplified TLS setup (for client and server)
    - Server middleware, including basic components for logging, token-based auth, rate limit, panic recovery, etc
    - Client middleware, including basic components for logging, custom metadata, panic recovery, etc
    - Bi-directional streaming support over upgraded HTTP(S) connections using WebSockets
    - Concurrent RPCs via connection pool

* [go.elara.ws/drpc](https://pkg.go.dev/go.elara.ws/drpc)
    - Concurrent RPCs based on [yamux](https://pkg.go.dev/github.com/hashicorp/yamux)
    - Simple drop-in replacements for `drpcserver` and `drpcconn`

 * Open an issue or join the [Zulip chat](https://drpc.zulipchat.com) if you'd like to be featured here.

 ## Examples

  * [A basic drpc client and server](../../tree/main/examples/drpc)
  * [A basic drpc client and server that also serves a Twirp/grpc-web compatible http server on the same port](../../tree/main/examples/drpc)
  * [Serving gRPC and DRPC on the same port](../../tree/main/examples/grpc_and_drpc)

## Other Languages

DRPC can be made compatible with RPC clients generated from other languages. For example, [Twirp](https://github.com/twitchtv/twirp) clients and [grpc-web](https://github.com/grpc/grpc-web/) clients can be used against the [drpchttp](https://pkg.go.dev/storj.io/drpc/drpchttp) package.

Native implementations can have some advantages, and so some support for other languages are in progress, all in various states of completeness. Join the [Zulip chat](https://drpc.zulipchat.com) if you want more information or to help out with any!

| Language | Repository                          | Status     |
|----------|-------------------------------------|------------|
| C++      | https://github.com/storj/drpc-cpp   | Incomplete |
| Rust     | https://github.com/zeebo/drpc-rs    | Incomplete |
| Node     | https://github.com/mjpitz/drpc-node | Incomplete |

## Licensing

DRPC is licensed under the MIT/expat license. See the LICENSE file for more.

---

## Benchmarks

These microbenchmarks attempt to provide a comparison and come with some caveats. First, it does not send data over a network connection which is expected to be the bottleneck almost all of the time. Second, no attempt was made to do the benchmarks in a controlled environment (CPU scaling disabled, noiseless, etc.). Third, no tuning was done to ensure they're both performing optimally, so there is an inherent advantage for DRPC because the author is familiar with how it works.

<table>
    <tr>
        <td rowspan=2>Measure</td>
        <td rowspan=2>Benchmark</td><td rowspan=2></td>
        <td colspan=3>Small</td><td rowspan=2></td>
        <td colspan=3>Medium</td><td rowspan=2></td>
        <td colspan=3>Large</td>
    </tr>
    <tr>
        <td>gRPC</td><td>DRPC</td><td>delta</td>
        <td>gRPC</td><td>DRPC</td><td>delta</td>
        <td>gRPC</td><td>DRPC</td><td>delta</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>time/op</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>29.7µs</td><td>8.3µs</td><td>-72.18%</td><td rowspan=4></td>
        <td>36.4µs</td><td>11.3µs</td><td>-68.92%</td><td rowspan=4></td>
        <td>1.70ms</td><td>0.54ms</td><td>-68.24%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>1.56µs</td><td>0.79µs</td><td>-49.07%</td>
        <td>3.80µs</td><td>2.04µs</td><td>-46.28%</td>
        <td>784µs</td><td>239µs</td><td>-69.48%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>1.51µs</td><td>0.78µs</td><td>-48.47%</td>
        <td>3.81µs</td><td>2.02µs</td><td>-47.06%</td>
        <td>691µs</td><td>224µs</td><td>-67.55%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>8.79µs</td><td>3.25µs</td><td>-63.07%</td>
        <td>13.7µs</td><td>5.0µs</td><td>-63.73%</td>
        <td>1.73ms</td><td>0.47ms</td><td>-72.72%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>speed</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>70.0kB/s</td><td>240.0kB/s</td><td>+242.86%</td><td rowspan=4></td>
        <td>56.3MB/s</td><td>181.1MB/s</td><td>+221.52%</td><td rowspan=4></td>
        <td>618MB/s</td><td>1939MB/s</td><td>+213.84%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>1.28MB/s</td><td>2.52MB/s</td><td>+96.11%</td>
        <td>540MB/s</td><td>1006MB/s</td><td>+86.16%</td>
        <td>1.34GB/s</td><td>4.38GB/s</td><td>+226.51%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>1.33MB/s</td><td>2.57MB/s</td><td>+93.88%</td>
        <td>538MB/s</td><td>1017MB/s</td><td>+89.14%</td>
        <td>1.52GB/s</td><td>4.68GB/s</td><td>+208.05%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>230kB/s</td><td>616kB/s</td><td>+167.93%</td>
        <td>149MB/s</td><td>412MB/s</td><td>+175.73%</td>
        <td>610MB/s</td><td>2215MB/s</td><td>+262.96%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>mem/op</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>9.42kB</td><td>1.42kB</td><td>-84.95%</td><td rowspan=4></td>
        <td>22.7kB</td><td>7.8kB</td><td>-65.61%</td><td rowspan=4></td>
        <td>6.42MB</td><td>3.16MB</td><td>-50.74%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>465B</td><td>80B</td><td>-82.80%</td>
        <td>7.06kB</td><td>2.13kB</td><td>-69.87%</td>
        <td>3.20MB</td><td>1.05MB</td><td>-67.10%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>360B</td><td>80B</td><td>-77.81%</td>
        <td>6.98kB</td><td>2.13kB</td><td>-69.52%</td>
        <td>3.20MB</td><td>1.05MB</td><td>-67.21%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>1.09kB</td><td>0.24kB</td><td>-77.94%</td>
        <td>14.4kB</td><td>4.3kB</td><td>-69.90%</td>
        <td>6.42MB</td><td>2.10MB</td><td>-67.22%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>allocs/op</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>182</td><td>7</td><td>-96.15%</td><td rowspan=4></td>
        <td>184</td><td>9</td><td>-95.11%</td><td rowspan=4></td>
        <td>280</td><td>9</td><td>-96.79%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>11</td><td>1</td><td>-90.91%</td>
        <td>12</td><td>2</td><td>-83.33%</td>
        <td>39.2</td><td>2</td><td>-94.90%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>11</td><td>1</td><td>-90.91%</td>
        <td>12</td><td>2</td><td>-83.33%</td>
        <td>38</td><td>2</td><td>-94.74%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>43</td><td>3</td><td>-93.02%</td>
        <td>46</td><td>5</td><td>-89.13%</td>
        <td>140</td><td>5</td><td>-96.43%</td>
    </tr>
</table>

## Lines of code

DRPC is proud to get as much done in as few lines of code as possible. It's the author's belief that this is only possible by having a clean, strong architecture and that it reduces the chances for bugs to exist (most studies show a linear corellation with number of bugs and lines of code). This table helps keep the library honest, and it would be nice if more libraries considered this.

| Package                              | Lines    |
| ---                                  | ---      |
| storj.io/drpc/drpcstream             | 486      |
| storj.io/drpc/drpchttp               | 478      |
| storj.io/drpc/cmd/protoc-gen-go-drpc | 428      |
| storj.io/drpc/drpcmanager            | 376      |
| storj.io/drpc/drpcwire               | 363      |
| storj.io/drpc/drpcpool               | 279      |
| storj.io/drpc/drpcmigrate            | 239      |
| storj.io/drpc/drpcserver             | 164      |
| storj.io/drpc/drpcconn               | 134      |
| storj.io/drpc/drpcsignal             | 133      |
| storj.io/drpc/drpcmetadata           | 115      |
| storj.io/drpc/drpcmux                | 95       |
| storj.io/drpc/drpccache              | 54       |
| storj.io/drpc                        | 47       |
| storj.io/drpc/drpctest               | 45       |
| storj.io/drpc/drpcerr                | 42       |
| storj.io/drpc/drpcctx                | 41       |
| storj.io/drpc/internal/drpcopts      | 30       |
| storj.io/drpc/drpcstats              | 25       |
| storj.io/drpc/drpcdebug              | 22       |
| storj.io/drpc/drpcenc                | 15       |
| **Total**                            | **3611** |
