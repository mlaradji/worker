# Job Scheduler

An API that allows authenticated clients to run arbitrary linux commands

## Build

To build the gRPC server and CLI, run `make build`. This should output the binaries `bin/worker-server` and `bin/worker-cli`.

To generate `pb`, run `make pb-clean` and then `make pb-gen`.

To generate certificates, run `make tls-clean` and then `make tls-gen`.

## Worker Client

The worker client can be started through either `go run cmd/cli/main.go`, or `./bin/worker-cli` if the binary was built. See `--help` for usage.

## Worker Server

The worker server can be started through either `go run cmd/server/main.go`, or `./bin/worker-server` if the binary was built. See `--help` for usage.

### Clients

There are 4 example client certificates that can be used. The server only accepts certificates signed by CA 1 for authentication. Clients 1, 2 and 3 were signed by CA 1, and client 4 by CA 2. Only Clients 1 and 2 are authorized to use the worker server.

## Worker Library

The worker library implements a job store that allows one to start, stop, log or query jobs. Clients can only query jobs that they have created.

For an example of the usage of the library, see `example/worker_library.go`.
