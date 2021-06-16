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
        <td>30.5µs</td><td>9.7µs</td><td>-68.10%</td><td rowspan=4></td>
        <td>38.5µs</td><td>12.3µs</td><td>-68.07%</td><td rowspan=4></td>
        <td>1.39ms</td><td>0.64ms</td><td>-53.89%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>862ns</td><td>835ns</td><td>~</td>
        <td>2.99µs</td><td>2.58µs</td><td>-13.68%</td>
        <td>504µs</td><td>259µs</td><td>-48.53%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>851ns</td><td>849ns</td><td>~</td>
        <td>2.82µs</td><td>2.53µs</td><td>-10.28%</td>
        <td>494µs</td><td>245µs</td><td>-50.43%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>9.84µs</td><td>3.68µs</td><td>-62.58%</td>
        <td>14.9µs</td><td>5.4µs</td><td>-63.91%</td>
        <td>1.34ms</td><td>0.56ms</td><td>-58.14%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>speed</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>70.0kB/s</td><td>207.5kB/s</td><td>+196.43%</td><td rowspan=4></td>
        <td>53.4MB/s</td><td>167.2MB/s</td><td>+213.23%</td><td rowspan=4></td>
        <td>753MB/s</td><td>1631MB/s</td><td>+116.64%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>2.32MB/s</td><td>2.40MB/s</td><td>~</td>
        <td>679MB/s</td><td>795MB/s</td><td>+17.06%</td>
        <td>2.08GB/s</td><td>4.04GB/s</td><td>+94.23%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>2.35MB/s</td><td>2.36MB/s</td><td>~</td>
        <td>729MB/s</td><td>812MB/s</td><td>+11.43%</td>
        <td>2.12GB/s</td><td>4.28GB/s</td><td>+101.68%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>200kB/s</td><td>540kB/s</td><td>+170.00%</td>
        <td>138MB/s</td><td>380MB/s</td><td>+176.07%</td>
        <td>785MB/s</td><td>1875MB/s</td><td>+138.88%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>mem/op</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>8.37kB</td><td>1.54kB</td><td>-81.63%</td><td rowspan=4></td>
        <td>21.8kB</td><td>7.9kB</td><td>-63.67%</td><td rowspan=4></td>
        <td>6.51MB</td><td>3.16MB</td><td>-51.42%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>398B</td><td>80B</td><td>-79.89%</td>
        <td>7.09kB</td><td>2.13kB</td><td>-70.01%</td>
        <td>3.20MB</td><td>1.05MB</td><td>-67.16%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>315B</td><td>80B</td><td>-74.61%</td>
        <td>6.99kB</td><td>2.13kB</td><td>-69.53%</td>
        <td>3.20MB</td><td>1.05MB</td><td>-67.16%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>1.02kB</td><td>0.24kB</td><td>-76.40%</td>
        <td>14.4kB</td><td>4.3kB</td><td>-69.99%</td>
        <td>6.52MB</td><td>2.10MB</td><td>-67.74%</td>
    </tr>
    <tr><td colspan=14></td></tr>
    <tr>
        <td rowspan=4>allocs/op</td>
        <td>Unitary</td><td rowspan=4></td>
        <td>169</td><td>14</td><td>-91.72%</td><td rowspan=4></td>
        <td>171</td><td>16</td><td>-90.64%</td><td rowspan=4></td>
        <td>402</td><td>16</td><td>-96.02%</td>
    </tr>
    <tr>
        <td>Input Stream</td>
        <td>11</td><td>1</td><td>-90.91%</td>
        <td>12</td><td>2</td><td>-83.33%</td>
        <td>119</td><td>2</td><td>-98.32%</td>
    </tr>
    <tr>
        <td>Output Stream</td>
        <td>9</td><td>1</td><td>-88.89%</td>
        <td>10</td><td>2</td><td>-80.00%</td>
        <td>118</td><td>2</td><td>-98.31%</td>
    </tr>
    <tr>
        <td>Bidir Stream</td>
        <td>41</td><td>3</td><td>-92.68%</td>
        <td>44</td><td>5</td><td>-88.64%</td>
        <td>280</td><td>5</td><td>-98.21%</td>
    </tr>
</table>

## Licensing

DRPC is licensed under the MIT/expat license. See the LICENSE file for more.
