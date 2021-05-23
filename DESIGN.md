Job Worker Service: Design Document

# Overview
The Job Worker project aims to implement a gRPC API that allows authenticated clients to run commands. The API is to run on a Linux server and should allow the client to execute any process that is available on the server. The project will be implemented in Go.

# Design Approach
## Components
### 1. Worker Library
With the worker library, a user can run a command, stop it, query its status and view its logs and output. The process will be executed as follows (in rough order):
1. The process is started in a dedicated process group by setting the PGID.
2. Three input channels are opened:
    - A channel of type `bool`, to listen for input indicating normal process end.
    - A channel of type `bool`, to listen for input indicating user-initiated stop request.
    - A channel of type `os.Signal`, to listen for OS signals. When a `SIGTERM` signal is received, a `SIGKILL` is sent to the process group to ensure all child processes are killed too.
3. Three output channels of type `string` are opened, which receive the final output/exit code, the STDOUT output, and the STDERR output, respectively.
### 2. gRPC API Daemon
### 3. Command-line Client
The command-line client connects to the gRPC daemon and allows the user to interact with the worker library. The suggested command-line interface is as follows (in [docopt](http://docopt.org/) format):
```
Usage:
  worker-cli [options] start -- <command>...
  worker-cli [options] (stop|status|output) <jobId>
  worker-cli [options] logs (all|stdout|stderr) <jobId> [--follow]
  worker-cli [options] list (all|running|failed|cancelled|done)
  worker-cli -h | --help
  worker-cli --version

Commands:
  start     
Options:
  -h --help             Show this screen.
  --version             Show version.
  --debug               Set log level to DEBUG.
  --address=<addr>      Server address and port [default: 0.0.0.0:8080]
  --cert=<cert>         Client certificate for mTLS. [default: cert/client-cert.pem]
  --key=<key>           Client key for mTLS. [default: cert/client-key.pem]
  --ca=<ca>             Certificate authority certificate for the server for mTLS. [default: cert/ca-cert.pem]
```
For example,
```bash
worker-cli start -- sh -c "/bin/bash" # Executes the command `sh -c "/bin/bash"` in a new job. This should print 
worker-cli 

## Trade-offs
1. The API does not sanitize the user's inputted commands before execution, and it does not sandbox the executed process in any way. This means that the user can purposefully or inadvertently cause severe damage to the API host.
2. The worker library will use in-memory storage to keep track of launched processes. This means potentially high RAM usage, and no persistence. In production, it would probably be best to use an external database.

# Milestones/
## 1. Implement interfaces
Create the schema for the various types of variables that the worker library will handle. As a start, there should be a “Command” interface, that has the name of the process to execute and its arguments, and a “Job” interface, with “Status”, “JobId”, “PID” (or some way of tracking the Linux process), “StartDate”, “FinishDate”, “Stdout”, “Stderr”, and “ExitStatus”. New interfaces might be added and current ones modified if needed for the later stages.
## 2. Implement the core worker library
The core worker library will be used by the API. Given a single command, it should be able to start and stop it, retrieve its status (started/finished/error), and retrieve its logs and final output. It should be able to wait on a process until its done or errored, and then fill in the Job object details and return it.
## 3. Implement job orchestration
The library should be able to handle multiple running jobs at the same time. This will be done through a global thread-safe map of string (jobId) to Job object. Each time a job is created, a goroutine will launch the core worker library implemented in 2, and continuously update the relevant Job object until its done.
## 4. Implement HTTPS API with basic authentication
The API will expose four paths over REST:
‘/health/’: GET path - health check.
‘/start/’: POST path with body request schema `{“command”: “string”, “args”: “[string]”}` and response schema `{“status”: “running | error”, “jobId”: “string”, “error”: “string | null”}`. Starts a command. Errors if unable to schedule the process.
‘/stop/’: POST path with body request schema `{“jobId”: “string”}` and response schema `{“status”: “success | error”, “error”: “string | null”}`. Stops a running command. Errors if the process is already stopped.
‘/status/{jobId}’: GET path with `jobId` path parameter and response schema `{“status”: “success | error”, “error”: “string | null”, “jobStatus”: “string | null”}`.
Additionally, the last 3 paths will only succeed if the request has the appropriate authentication headers.
## 5. Implement client library
Expose the same paths in the previous milestone as shell commands: `--health`, `--start`, `--stop`, `--status`.
## 6. Implement tests
Add a test that runs a quick command with known output (such as `cat test`), checks that the response schema/object is valid, grab the status of the job and check the response.
## 7. Implement mTLS authentication
## 8. Implement GRPC API
## 9. Implement streaming of logs for client and GRPC

## Edge Cases

