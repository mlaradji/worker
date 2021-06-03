# Job Scheduler
An API that allows authenticated clients to run arbitrary linux commands

## Build
To build the gRPC server, run `make build`. This should output the binary `bin/worker-server` binary.

# Worker Library
The worker library implements a job store that allows one to start, stop, log or query jobs. Clients can only query jobs that they have created.

For an example of the usage of the library, see `example/worker_library.go`.