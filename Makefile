.PHONY: test
test:
	go test -race ./... -v

.PHONY: run 
run:
	go run main.go

.PHONY: compile
compile:
	go build -o ./bin/monkeyd . && ./bin/monkeyd

