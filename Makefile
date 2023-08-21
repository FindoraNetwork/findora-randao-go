all: rebuild

rebuild: clean build

build: compile abigen build_debug

compile:
	solc contract/IRandao.sol --abi --bin -o ./abi

abigen: 
	abigen --abi abi/IRandao.abi --bin abi/IRandao.bin --pkg randao --type randao  --out contract/randao.go

build_debug:
	go build -tags "debug" -o ./bin/participant ./cmd/participant
	go build -tags "debug" -o ./bin/campaign ./cmd/campaign

build_release:
	go build -o ./bin/participant ./cmd/participant
	go build -o ./bin/campaign ./cmd/campaign

clean:
	rm bin/* abi/* contract/randao.go -rf
	go clean -testcache 
	
fmt:
	go fmt ./...

test:
	go test -v ./...

