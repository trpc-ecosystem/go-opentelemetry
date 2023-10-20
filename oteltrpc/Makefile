# log、metrics、trace benchmark test
benchmark:
	go test ./logs ./metrics/prometheus ./traces -bench=. -run=^$$ -benchtime=5s > ./docs/benchmark.txt
