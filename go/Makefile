BIN_NAME = e2json

test:
	@echo "run unit-tests"
	@go test *.go

build: test
	@echo "build"
	@CGO_ENABLE=0 go build -ldflags '-s -w' -o bin/$(BIN_NAME) exec.go
