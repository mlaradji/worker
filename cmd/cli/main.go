package main

import (
	"context"
	"fmt"
	"io"

	"github.com/docopt/docopt-go"
	"github.com/mlaradji/int-backend-mohamed/pb"
	"github.com/mlaradji/int-backend-mohamed/service"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Usage is the help docs, which docopt can directly parse.
const Usage = `Usage:
	worker-cli [options] start -- <command> [<args>...]
	worker-cli [options] (stop|status|logs) <jobId>
	worker-cli -h | --help
	worker-cli --version

Options:
	-h --help             Show this screen.
	--version             Show version.
	--debug               Set log level to DEBUG.
	--address=<addr>      Server address and port [default: 0.0.0.0:8000]
	--cert=<cert>         Path to the client certificate for mTLS. [default: certs/client1/cert.pem]
	--key=<key>           Path to the client key for mTLS. [default: certs/client1/key.pem]
	--ca=<ca>             Path to the CA certificate for the server for mTLS. [default: certs/ca1/cert.pem]

Commands:
	start     Start a new job for the input command. If successful, the new job id will be printed.
	stop      Stop a job. No error is emitted if job is already done or stopped.
	status    Query the status and other information of a job. The status of a job is one of created|running|succeeded|failed|stopped.
	logs      Follow logs (STDOUT+STDERR) of a job.`

// Configuration contains all variables that were passed (implicity or explicitly) to the command.
type Configuration struct {
	// options

	Debug   bool   `docopt:"--debug"`
	Address string `docopt:"--address"`
	Cert    string `docopt:"--cert"`
	Key     string `docopt:"--key"`
	CA      string `docopt:"--ca"`

	// chosen sub-command

	Start  bool `docopt:"start"`
	Logs   bool `docopt:"logs"`
	Status bool `docopt:"status"`
	Stop   bool `docopt:"stop"`

	// start job

	DashDash bool     `docopt:"--"`
	Command  string   `docopt:"<command>"`
	Args     []string `docopt:"<args>"`

	// other commands

	JobId string `docopt:"<jobId>"`
}

var (
	Config         = &Configuration{}
	TLSCredentials credentials.TransportCredentials
)

func init() {
	logger := log.WithField("func", "init")

	opts, err := docopt.ParseDoc(Usage)
	if err != nil {
		logger.WithError(err).Fatal("unable to parse usage doc")
	}

	// extract config fields
	err = opts.Bind(Config)
	if err != nil {
		logger.WithError(err).Fatal("unable to type cast configuration")
	}

	// enable debug logs if --debug was passed
	if Config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	logger.WithField("Config", Config).Debug("successfully parsed configuration")

	// load certificates
	cert, certPool, err := service.LoadTLSCertificate(Config.CA, Config.Cert, Config.Key)
	if err != nil {
		logger.WithError(err).Fatal("unable to load TLS certificate")
	}

	TLSCredentials = service.MakeClientTLSCredentials(cert, certPool)

	logger.Debug("successfully loaded certificates")
}

func main() {
	logger := log.WithField("func", "main")

	conn, err := grpc.Dial(Config.Address, grpc.WithTransportCredentials(TLSCredentials))
	if err != nil {
		logger.WithError(err).Fatal("cannot dial server")
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			logger.WithError(err).Error("unable to close connection")
		}
	}()

	ctx := context.Background()
	client := pb.NewJobServiceClient(conn)

	logger.Debug("successfully initialized client")

	if Config.Start {
		// start a new job
		res, err := client.JobStart(ctx, &pb.JobStartRequest{Command: Config.Command, Args: Config.Args})
		if err != nil {
			logger.WithError(err).Fatal("received an error response")
		}

		logger.WithField("jobId", res.GetJobId()).Info("job was started successfully")
		fmt.Printf("JobId: %s", res.GetJobId())
		return
	}

	if Config.Stop {
		// stop a current job
		_, err := client.JobStop(ctx, &pb.JobStopRequest{JobId: Config.JobId})
		if err != nil {
			logger.WithError(err).Fatal("received an error response")
		}

		logger.Info("job stop request successfully sent")
		return
	}

	if Config.Status {
		// query an existing job's status
		res, err := client.JobStatus(ctx, &pb.JobStatusRequest{JobId: Config.JobId})
		if err != nil {
			logger.WithError(err).Fatal("received an error response")
		}

		logger.Debug("job status successfully queried")
		fmt.Printf("status: %s", res.GetJobInfo())
		return
	}

	if Config.Logs {
		// follow a job's logs
		logStream, err := client.JobLogsStream(ctx, &pb.JobLogsRequest{JobId: Config.JobId})
		if err != nil {
			logger.WithError(err).Fatal("received an error response")
		}

		for {
			logRes, err := logStream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.WithError(err).Fatal("failed while streaming logs")
			}

			fmt.Print(logRes.GetLog())
		}

		logger.Info("done streaming logs - job is not running")
		return
	}
}
