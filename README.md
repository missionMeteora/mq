# MQ 
MQ is a simple message queue written in Go

# Benchmarks
```
BenchmarkMQ_32B-4         	 2000000        1136 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_32B-4     	  200000        8261 ns/op           65 B/op       5 allocs/op

BenchmarkMQ_128B-4        	  500000        3305 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_128B-4    	  200000        6899 ns/op          163 B/op       5 allocs/op

BenchmarkMQ_1KB-4         	  500000        3740 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_1KB-4     	  200000        5593 ns/op         1156 B/op       5 allocs/op

BenchmarkMQ_4KB-4         	 1000000        3341 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_4KB-4     	  200000        7218 ns/op         5069 B/op       5 allocs/op

BenchmarkMQ_16KB-4        	  300000        5549 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_16KB-4    	  200000       11669 ns/op        23368 B/op       5 allocs/op

BenchmarkMQ_32KB-4        	  200000        7635 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_32KB-4    	  100000       19598 ns/op        44318 B/op       5 allocs/op

BenchmarkMQ_64KB-4        	  100000       12514 ns/op           32 B/op       1 allocs/op
BenchmarkMangos_64KB-4    	   30000       48588 ns/op       197128 B/op      11 allocs/op

BenchmarkMQ_256KB-4       	   50000       33951 ns/op           37 B/op       1 allocs/op
BenchmarkMangos_256KB-4   	   10000      163763 ns/op       786875 B/op      11 allocs/op

BenchmarkMQ_512KB-4       	   20000       62970 ns/op           58 B/op       1 allocs/op
BenchmarkMangos_512KB-4   	    5000      334432 ns/op      1573387 B/op      11 allocs/op

BenchmarkMQ_1MB-4         	   10000      158345 ns/op          136 B/op       1 allocs/op
BenchmarkMangos_1MB-4     	    2000      752500 ns/op      3146674 B/op      11 allocs/op
```