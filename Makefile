pb-gen:
	protoc --proto_path=proto proto/*.proto  --go_out=paths=source_relative:pb --go-grpc_out=paths=source_relative:pb

pb-clean:
	rm pb/* -r

clean:
	rm tmp/* -r
	
certs:
	cd certs && ./generate.sh

test:
	go test -v -cover -race -timeout 30s ./...

certs-clean:
	rm -r certs/**/*.pem

server:
	go run cmd/server/main.go

.PHONY: certs server
