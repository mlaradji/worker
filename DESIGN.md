Job Worker Service: Design Document

# Overview
The Job Worker project aims to implement a gRPC API that allows authenticated clients to run commands. The API is to run on a Linux server and should allow the client to execute any process that is available on the server. The project will be implemented in Go.

# Design Approach
## Components
### 1. Worker Library
With the worker library, a user can run a command, stop it, query its status and info and view its logs and output, and list all jobs.

The library stores job information in an in-memory job id to job object map, the Job Store. The job object contains job information, as well as job-related channels. The job id is a randomly generated UUIDv4.

When the library is initialized, a Job Store and a message bus is created.

When the worker library receives a request to run a command (by calling the appropriate library method), it starts an Executing Thread that executes the command in a dedicated process group by setting the PGID. The new thread opens several channels to listen for input indicating normal process end, a user-initiated stop request, and OS signals. The thread publishes the exit code, the STDOUT output, and the STDERR output on dedicated message bus channels (`'<jobId>:output'`, `'<jobId>:stdout'`, `'<jobId>:stderr'`), and stores the log data in the Job Store along with the job status and the input channel objects.

Reading a job's log is done directly from the Job Store. Streaming logs is done by subscribing to the appropriate message bus channel. 

When the worker library receives a request to stop a job, it is forwarded to the Executing Thread of that job through the stop request channel. The Executing Thread then sends a `SIGKILL` signal to the process group using the PGID to ensure the job and all child processes are killed too.

### 2. gRPC API Daemon
The daemon acts as an interface to the worker library, and exposes all of its functions over gRPC. The daemon also handles user authentication and authorization. The service definitions can be found in `proto/`.

### 3. Command-line Client
The command-line client connects to the gRPC daemon and allows the user to interact with the worker library. The suggested command-line interface is as follows (in [docopt](http://docopt.org/) format):
```
Usage:
  worker-cli [options] start -- <command>...
  worker-cli [options] (stop|status) <jobId>
  worker-cli [options] logs (stdout|stderr) <jobId>
  worker-cli [options] list
  worker-cli -h | --help
  worker-cli --version

Options:
  -h --help             Show this screen.
  --version             Show version.
  --debug               Set log level to DEBUG.
  --address=<addr>      Server address and port [default: 0.0.0.0:8080]
  --cert=<cert>         Path to the client certificate for mTLS. [default: cert/client-cert.pem]
  --key=<key>           Path to the client key for mTLS. [default: cert/client-key.pem]
  --ca=<ca>             Path to the CA certificate for the server for mTLS. [default: cert/ca-cert.pem]

Commands:
  start     Start a new job for the input command. If successful, the new job id will be printed.
  stop      Stop a job. No error is emitted if job is already done or stopped.
  status    Query the status and other information of a job. The status of a job is one of running|succeeded|failed|stopped.
  logs      Follow STDOUT or STDERR logs of a job.
  list      List all jobs.
```
For example,
```bash
# Executes the command `sh -c "/bin/bash"` in a new job. This prints the new job id.
worker-cli start -- sh -c "/bin/bash" # output: 5a2e

# Stop the job with id 5a2e. This blocks until the job is no longer running, and outputs either an error or the job's current status.
worker-cli stop 5a2e # output: stopped

# Stream STDOUT logs for 5a2e. This is blocking if job is still running.
worker-cli logs stdout 5a2e --follow
```

### Authentication
The project uses mTLS, and only TLS 1.3 is accepted for authentication. The allowed cipher suites are:
```
	TLS_CHACHA20_POLY1305_SHA256
	TLS_AES_256_GCM_SHA384
	TLS_AES_128_GCM_SHA256
```

The projects uses X.509v3 certificates, with 4096-bit RSA encryption, SHA256 signature, and the X.509v3 Subject Alternative Name extension. A new self-signed Certificate Authority will be created solely for the project, and the server and client certificates will be newly created and signed by the CA. All certificates and keys will be stored unencrypted and pushed to the repository.

### Authorization
Authorization relies on the SHA256 fingerprint of client certificates. After a client successfully authenticates, their entire raw certificate is hashed through SHA256 to produce a fingerprint. The fingerprint is checked against a hard-coded table of roles and fingerprints. The available roles are:
- `LOG`: allows the user to list jobs and query their status,
- `FULL`: allows the user full access to all functions of the API.

For example, with the following role table,
```
FULL:
  - aaaaaaaaaaaaaaaa # fingerprint of client A cert
  - bbbbbbbbbbbbbbbb # fingerprint of client B cert

LOG:
  - cccccccccccccccc # fingerprint of client C cert
```
clients A and B are given full access to the API, while client C is only allowed to list jobs and query their status. Clients with fingerprint not in either role will not have any access to the API. Clients in both roles will have the `FULL` role.

## Trade-offs
1. The API does not sanitize the user's inputted commands before execution, and it does not sandbox the executed process in any way. This means that the user can purposefully or inadvertently cause severe damage to the API host.
2. The worker library uses in-memory storage to keep track of launched processes. This means potentially high RAM usage and no persistence. In production, it would probably be best to use an external database.
3. The gRPC daemon only accepts TLS 1.3 ciphers for encryption and authentication. This choice might affect client compatibility.
4. For mTLS authorization, a hard-coded list of client signatures and roles will be used. Ideally, the server should either allow an administrator user to add and remove signatures and roles, or rely on a third-party authorization server.
5. The mTLS certificate authority will be self-signed, the certificates will be created and stored locally, and all keys and certificates will be unencrypted and pushed to the repository. This is a security risk.

## Edge Cases
1. Starting too many jobs too quickly can cause the OS to spend a lot of time on system calls.
2. If the CLI is used to run another instance of the CLI that runs a command, stopping the job may not work as expected. Similarly, the CLI could be used to stop the server, which might cause orphan threads.

# Milestones
## 1. Implement the worker library with tests
## 2. Implement the gRPC server with tests
## 3. Implement the gRPC CLI with tests

