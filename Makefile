pb-gen:
	protoc --proto_path=proto proto/*.proto  --go_out=paths=source_relative:pb --go-grpc_out=paths=source_relative:pb

pb-clean:
	rm pb/* -r

clean:
	rm tmp/* -r

test:
	go test -v -cover -race -timeout 5s ./...
