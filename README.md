# Job Scheduler

An API that allows authenticated clients to run arbitrary linux commands

## Build

To build the gRPC server, run `make build`. This should output the binary `bin/worker-server` binary.

## Worker Server

The worker server can be started through either `go run cmd/server/main.go`, or `./bin/worker-server` if the binary was built. See `--help` for usage.

### Clients

There are 4 example client certificates that can be used. The server only accepts certificates signed by CA 1 for authentication. Clients 1, 2 and 3 were signed by CA 1, and client 4 by CA 2. Only Clients 1 and 2 are authorized to use the worker server.

## Worker Library

The worker library implements a job store that allows one to start, stop, log or query jobs. Clients can only query jobs that they have created.

For an example of the usage of the library, see `example/worker_library.go`.
