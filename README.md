# MQ 
MQ is a simple message queue written in Go

# Benchmarks
```
BenchmarkMQ_32B-4                5000000              1227 ns/op              32 B/op          1 allocs/op
BenchmarkMQ_128B-4              10000000              1352 ns/op              32 B/op          1 allocs/op
BenchmarkMQ_1KB-4                1000000              6116 ns/op              32 B/op          1 allocs/op
BenchmarkMQ_4KB-4                2000000              6588 ns/op              32 B/op          1 allocs/op
BenchmarkMQ_16KB-4               1000000              6024 ns/op              32 B/op          1 allocs/op
BenchmarkMQ_1MB-4                 100000            123447 ns/op              73 B/op          1 allocs/op

BenchmarkMangos_32B-4            1000000              7743 ns/op              65 B/op          5 allocs/op
BenchmarkMangos_128B-4           1000000              7550 ns/op             162 B/op          5 allocs/op
BenchmarkMangos_1KB-4            1000000              7865 ns/op            1128 B/op          5 allocs/op
BenchmarkMangos_4KB-4            1000000              8072 ns/op            4993 B/op          5 allocs/op
BenchmarkMangos_16KB-4           1000000             11707 ns/op           23114 B/op          5 allocs/op
BenchmarkMangos_1MB-4              10000            746335 ns/op         3146250 B/op         11 allocs/op
```