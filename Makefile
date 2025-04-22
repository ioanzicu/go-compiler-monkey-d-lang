.PHONY: test
test:
	go test -race ./... -v

.PHONY: run 
run:
	go run main.go

.PHONY: compile
compile:
	go build -o ./bin/monkeyd . && ./bin/monkeyd

.PHONY: benchmark
benchmark:
	go build -o fibonacci ./benchmark

.PHONY: benchmark-eval
benchmark-eval: benchmark
	./fibonacci -engine=eval

.PHONY: benchmark-vm 
benchmark-vm: benchmark
	./fibonacci -engine=vm