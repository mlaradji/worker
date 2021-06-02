cli:
	go run cmd/cli/main.go

server:
	go run cmd/server/main.go

build:
	mkdir -p bin
	go build -o bin/worker-cli cmd/cli/main.go
	go build -o bin/worker-server cmd/server/main.go

test:
	go test -v -cover -race -timeout 30s ./...

clean:
	rm tmp/* -r

tls-gen:
	cd certs
	./generate.sh

tls-clean:
	rm -r certs/**/*.pem

pb-gen:
	protoc --proto_path=proto proto/*.proto  --go_out=paths=source_relative:pb --go-grpc_out=paths=source_relative:pb

pb-clean:
	rm pb/* -r

.PHONY: certs
