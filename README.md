# [![DRPC](logo.png)](https://storj.github.io/drpc/)

A drop-in, lightweight gRPC replacement.

[![Go Report Card](https://goreportcard.com/badge/storj.io/drpc)](https://goreportcard.com/report/storj.io/drpc)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/storj.io/drpc)
![Beta](https://img.shields.io/badge/version-beta-green.svg)

## Links

 * [DRPC website](https://storj.github.io/drpc/)
 * [Quickstart documentation](https://storj.github.io/drpc/docs.html)
 * [Launch blog post](https://www.storj.io/blog/introducing-drpc-our-replacement-for-grpc)
 * [Examples](https://github.com/storj/drpc/tree/main/examples)

## Highlights

* Simple, at just a few thousands lines of code!
* Compatible. Works for many gRPC use-cases as-is!
* Fast. DRPC has a lightning quick wire format
* Extensible. DRPC is transport agnostic, supports middleware, and is designed around interfaces.
* Battle Tested. Already used in production for years across tens of thousands of servers.

## Benchmarks

These microbenchmarks attempt to provide a comparison and come with some caveats. First, gRPC and DRPC have different flushing semantics when sending messages. Specifically, gRPC will buffer for some period whereas DRPC always immediately flushes. This difference is most apparent in the Input/Output Stream benchmarks under the Small and Medium sizes. Second, all microbenchmarks are flawed. These, in particular, do not even send data over a network connection. Third, no attempt was made to do the benchmarks in a controlled environment (CPU scaling disabled, noiseless, etc.).

<table>
    <tr>
        <td rowspan=2>Benchmark</td>
        <td rowspan=2>Measure</td><td rowspan=2></td>
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
        <td rowspan=4>Unitary</td>
        <td>time/op</td><td rowspan=4>
        <td>39.5µs</td><td>17.1µs</td><td><font color=green>-56.64%</font></td><td rowspan=4></td>
        <td>39.7µs</td><td>21.5µs</td><td><font color=green>-45.83%</font></td><td rowspan=4></td>
        <td>1.35ms</td><td>0.65ms</td><td><font color=green>-52.07%</font>
    </tr>
    <tr>
        <td>speed</td>
        <td>53.8kB/s</td><td>120.0kB/s</td><td><font color=green>+123.26%</font></td>
        <td>51.7MB/s</td><td>95.4MB/s</td><td><font color=green>+84.48%</font></td>
        <td>775MB/s</td><td>1618MB/s</td><td><font color=green>+108.64%</font>
    </tr>
    <tr>
        <td>mem/op</td>
        <td>8.43kB</td><td>2.05kB</td><td><font color=green>-75.65%</font></td>
        <td>21.9kB</td><td>8.5kB</td><td><font color=green>-61.43%</font></td>
        <td>6.51MB</td><td>3.22MB</td><td><font color=green>-50.44%</font>
    </tr>
    <tr>
        <td>allocs/op</td>
        <td>169</td><td>20</td><td><font color=green>-88.17%</font></td>
        <td>171</td><td>22</td><td><font color=green>-87.13%</font></td>
        <td>422</td><td>23</td><td><font color=green>-94.64%</font></td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>Input Stream</td>
        <td>time/op</td><td rowspan=4></td>
        <td>856ns</td><td>2501ns</td><td><font color=red>+192.29%</font></td><td rowspan=4></td>
        <td>2.95µs</td><td>3.37µs</td><td><font color=red>+14.09%</font></td><td rowspan=4></td>
        <td>544µs</td><td>263µs</td><td><font color=green>-51.73%</font></td>
    </tr>
    <tr>
        <td>speed</td>
        <td>2.28MB/s</td><td>0.80MB/s</td><td><font color=red>-64.91%</font></td>
        <td>696MB/s</td><td>610MB/s</td><td><font color=red>-12.35%</font></td>
        <td>1.93GB/s</td><td>3.99GB/s</td><td><font color=green>+107.18%</font></td>
    </tr>
    <tr>
        <td>mem/op</td>
        <td>409B</td><td>80B</td><td><font color=green>-80.46%</font></td>
        <td>7.09kB</td><td>2.13kB</td><td><font color=green>-69.99%</font></td>
        <td>3.22MB</td><td>1.08MB</td><td><font color=green>-66.39%</font></td>
    </tr>
    <tr>
        <td>allocs/op</td>
        <td>11</td><td>1</td><td><font color=green>-90.91%</font></td>
        <td>12</td><td>2</td><td><font color=green>-83.33%</font></td>
        <td>128</td><td>2</td><td><font color=green>-98.44%</font></td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>Output Stream</td>
        <td>time/op</td><td rowspan=4></td>
        <td>953ns</td><td>2585µs</td><td><font color=red>+171.35%</font></td><td rowspan=4></td>
        <td>2.87µs</td><td>3.49µs</td><td><font color=red>+21.56%</font></td><td rowspan=4></td>
        <td>532µs</td><td>247µs</td><td><font color=green>-53.50%</font></td>
    </tr>
    <tr>
        <td>speed</td>
        <td>4.20MB/s</td><td>1.55MB/s</td><td><font color=red>-63.15%</font></td>
        <td>716MB/s</td><td>589MB/s</td><td><font color=red>-17.74%</font></td>
        <td>1.97GB/s</td><td>4.24GB/s</td><td><font color=green>+115.02%</font></td>
    </tr>
    <tr>
        <td>mem/op</td>
        <td>371B</td><td>160B</td><td><font color=green>-56.89%</font></td>
        <td>7.06kB</td><td>2.21kB</td><td><font color=green>-68.75%</font></td>
        <td>3.21MB</td><td>1.06MB</td><td><font color=green>-66.98%</font></td>
    </tr>
    <tr>
        <td>allocs/op</td>
        <td>11</td><td>2</td><td><font color=green>-80.00%</font></td>
        <td>11</td><td>3</td><td><font color=green>-72.73%</font></td>
        <td>131</td><td>3</td><td><font color=green>-97.70%</font></td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>Bidir Stream</td>
        <td>time/op</td><td rowspan=4></td>
        <td>10.7µs</td><td>5.3µs</td><td><font color=green>-50.67</font></td><td rowspan=4></td>
        <td>15.9µs</td><td>7.2µs</td><td><font color=green>-54.36%</font></td><td rowspan=4></td>
        <td>1.38ms</td><td>0.61ms</td><td><font color=green>-55.79%</font></td>
    </tr>
    <tr>
        <td>speed</td>
        <td>185kB/s</td><td>379kB/s</td><td><font color=green>+104.63%</font></td>
        <td>129MB/s</td><td>284MB/s</td><td><font color=green>+119.11%</font></td>
        <td>761MB/s</td><td>1659MB/s</td><td><font color=green>+117.91%</font></td>
    </tr>
    <tr>
        <td>mem/op</td>
        <td>1.02kB</td><td>0.24kB</td><td><font color=green>-76.44%</font></td>
        <td>14.5kB</td><td>4.3kB</td><td><font color=green>-69.99%</font></td>
        <td>6.52MB</td><td>2.17MB</td><td><font color=green>-66.66%</font></td>
    </tr>
    <tr>
        <td>allocs/op</td>
        <td>41</td><td>3</td><td><font color=green>-92.68%</font></td>
        <td>44</td><td>5</td><td><font color=green>-88.64%</font></td>
        <td>291</td><td>6</td><td><font color=green>-98.07%</font></td>
    </tr>
</table>

## Licensing

DRPC is licensed under the MIT/expat license. See the LICENSE file for more.
