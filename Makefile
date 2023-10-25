all: rebuild

rebuild: clean build_debug

rebuild_release: clean build_release

compile:
	solc contract/randao/eth/contracts/IRandao.sol --abi --bin -o ./abi

abigen: 
	abigen --abi abi/IRandao.abi --bin abi/IRandao.bin --pkg contract --type randao  --out contract/randao.go

build_debug: compile abigen
	go build -tags "debug" -o ./bin/participant ./cmd/participant
	go build -tags "debug" -o ./bin/campaign ./cmd/campaign

build_release: compile abigen
	go build -o ./bin/participant ./cmd/participant
	go build -o ./bin/campaign ./cmd/campaign

clean:
	rm bin/* abi/* contract/randao.go -rf
	go clean -testcache 
	
fmt:
	go fmt ./...

test:
	go test -v ./...

