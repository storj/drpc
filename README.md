# [![DRPC](logo.png)](https://storj.github.io/drpc/)

A drop-in, lightweight gRPC replacement.

[![Go Report Card](https://goreportcard.com/badge/storj.io/drpc)](https://goreportcard.com/report/storj.io/drpc)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/storj.io/drpc)
![Beta](https://img.shields.io/badge/version-beta-green.svg)

Please see our:

 * [DRPC website](https://storj.github.io/drpc/)
 * [Quickstart documentation](https://storj.github.io/drpc/docs.html)
 * [Launch blog post](https://www.storj.io/blog/introducing-drpc-our-replacement-for-grpc)

DRPC is:

* Simple, at just a few thousands lines of code!
* Compatible. Works for many gRPC use-cases as-is!
* Fast. DRPC has a lightning quick wire format
* Extensible. DRPC is transport agnostic, supports middleware, and is designed around interfaces.

Compare with gRPC:

```
BenchmarkUnitary/GRPC-8          	   40166	     29717 ns/op	    8271 B/op	     169 allocs/op
BenchmarkUnitary/DRPC-8          	   53380	     21705 ns/op	    2882 B/op	      39 allocs/op
BenchmarkInputStream/GRPC-8      	 1308865	       836.5 ns/op	     370 B/op	      11 allocs/op
BenchmarkInputStream/DRPC-8      	  449756	      2555 ns/op	      63 B/op	       3 allocs/op
BenchmarkOutputStream/GRPC-8     	 1453718	       910.2 ns/op	     283 B/op	       9 allocs/op
BenchmarkOutputStream/DRPC-8     	  455004	      2571 ns/op	      63 B/op	       3 allocs/op
BenchmarkBidirectionalStream/GRPC-8       113724	     10273 ns/op	     873 B/op	      40 allocs/op
BenchmarkBidirectionalStream/DRPC-8       218518	      5396 ns/op	     128 B/op	       6 allocs/op
```

## Licensing

DRPC is licensed under the MIT/expat license. See the LICENSE file for more.
