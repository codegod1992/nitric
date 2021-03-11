package membrane

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	grpc2 "github.com/nitric-dev/membrane/adapters/grpc"
	"github.com/nitric-dev/membrane/handler"

	v1 "github.com/nitric-dev/membrane/interfaces/nitric/v1"
	"github.com/nitric-dev/membrane/sdk"
	"google.golang.org/grpc"
)

type MembraneOptions struct {
	ServiceAddress string
	// The address the child will be listening on
	ChildAddress string
	// The command that will be used to invoke the child process
	ChildCommand string
	// The total time to wait for the child process to be available in seconds
	ChildTimeoutSeconds int

	EventingPlugin  sdk.EventService
	DocumentsPlugin sdk.DocumentService
	StoragePlugin   sdk.StorageService
	QueuePlugin     sdk.QueueService
	GatewayPlugin   sdk.GatewayService
	AuthPlugin      sdk.UserService

	SuppressLogs            bool
	TolerateMissingServices bool

	// The operating mode of the membrane
	Mode Mode
}

type Membrane struct {
	// Address & port to bind the membrane i/o proxy to
	// This will still be bound even in pass through mode
	// proxyAddress string
	// Address & port to bind the membrane service interfaces to
	serviceAddress string
	// The address the child will be listening on
	childAddress string

	// The URL (including protocol, the child process can be reached on)
	childUrl string

	// The command that will be used to invoke the child process
	childCommand string

	childTimeoutSeconds int

	// Configured plugins
	eventPlugin     sdk.EventService
	documentsPlugin sdk.DocumentService
	storagePlugin   sdk.StorageService
	gatewayPlugin   sdk.GatewayService
	queuePlugin     sdk.QueueService
	authPlugin      sdk.UserService

	// Tolerate if adapters are not available
	// Not this does not include the gateway service
	tolerateMissingServices bool

	// Suppress println statements in the membrane server
	supressLogs bool

	mode Mode
}

func (s *Membrane) log(log string) {
	if !s.supressLogs {
		fmt.Println(log)
	}
}

// Create a new Nitric Eventing Server
func (s *Membrane) createEventingServer() v1.EventServer {
	return grpc2.NewEventServer(s.eventPlugin)
}

func (s *Membrane) createTopicServer() v1.TopicServer {
	return grpc2.NewTopicServer(s.eventPlugin)
}

// Create a new Nitric Storage Server
func (s *Membrane) createStorageServer() v1.StorageServer {
	return grpc2.NewStorageServer(s.storagePlugin)
}

func (s *Membrane) createDocumentsServer() v1.DocumentServer {
	return grpc2.NewDocumentsServer(s.documentsPlugin)
}

func (s *Membrane) createQueueServer() v1.QueueServer {
	return grpc2.NewQueueServer(s.queuePlugin)
}

func (s *Membrane) createUserServer() v1.UserServer {
	return grpc2.NewUserServer(s.authPlugin)
}

func (s *Membrane) startChildProcess() error {
	// TODO: This is a detached process
	// so it will continue to run until even after the director dies
	commandArgs := strings.Fields(s.childCommand)

	fmt.Println(fmt.Sprintf("Starting Function"))
	childProcess := exec.Command(commandArgs[0], commandArgs[1:len(commandArgs)]...)
	childProcess.Stdout = os.Stdout
	childProcess.Stderr = os.Stderr
	applicationError := childProcess.Start()

	// Actual panic here, we don't want to start if our userland code cannot successfully start
	if applicationError != nil {
		return fmt.Errorf("There was an error starting the child process: %v", applicationError)
	}

	// Dial the child port to see if it's open and ready...
	// Only wait for 10s, if we timeout that will be it
	// TODO: make app startup time configurable
	maxWaitTime := time.Duration(s.childTimeoutSeconds) * time.Second
	// Longer poll times, e.g. 200 milliseconds results in slow lambda cold starts (15s+)
	pollInterval := time.Duration(15) * time.Millisecond

	var waitedTime = time.Duration(0)
	for {
		conn, _ := net.Dial("tcp", s.childAddress)
		if conn != nil {
			conn.Close()
			break
		} else {
			if waitedTime < maxWaitTime {
				time.Sleep(pollInterval)
				waitedTime += pollInterval
			} else {
				return fmt.Errorf("Unable to dial child server, does it expose a http server at: %s?", s.childAddress)
			}
		}
	}

	return nil
}

func (s *Membrane) nitricResponseFromError(err error) *sdk.NitricResponse {
	return &sdk.NitricResponse{
		Headers: map[string]string{"Content-Type": "text/plain"},
		Body:    []byte(err.Error()),
		Status:  503,
	}
}

// Start the membrane
func (s *Membrane) Start() error {
	// Search for known plugins

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	// Load & Register the GRPC service plugins
	eventingServer := s.createEventingServer()
	v1.RegisterEventServer(grpcServer, eventingServer)

	topicServer := s.createTopicServer()
	v1.RegisterTopicServer(grpcServer, topicServer)

	documentsServer := s.createDocumentsServer()
	v1.RegisterDocumentServer(grpcServer, documentsServer)

	storageServer := s.createStorageServer()
	v1.RegisterStorageServer(grpcServer, storageServer)

	queueServer := s.createQueueServer()
	v1.RegisterQueueServer(grpcServer, queueServer)

	authServer := s.createUserServer()
	v1.RegisterUserServer(grpcServer, authServer)

	lis, err := net.Listen("tcp", s.serviceAddress)
	if err != nil {
		return fmt.Errorf("Could not listen on configured service address: %v", err)
	}

	s.log("Registered Gateway Plugin")

	// Start the gRPC server
	go (func() {
		s.log(fmt.Sprintf("Services listening on: %s", s.serviceAddress))
		grpcServer.Serve(lis)
	})()

	// Start our child process
	// This will block until our child process is ready to accept incoming connections
	if s.childCommand != "" {
		if err := s.startChildProcess(); err != nil {
			// Return the error
			return err
		}
	} else {
		s.log("No Child Configured Specified, Skipping...")
	}

	// FIXME: Only do this in Gateway mode...
	// Otherwise always pass through to the provided child address
	// Start the Gateway Server

	// Start the gateway, this will provide us an entrypoint for
	// data ingress/egress to our userland code
	// The gateway should block the main thread but will
	// use this callback as a control mechanism
	s.log("Starting Gateway")

	var hndlr handler.SourceHandler
	switch s.mode {
	case Mode_Faas:
		hndlr = handler.NewFaasHandler(s.childAddress)
		break
	case Mode_Http:
		hndlr = handler.NewHttpHandler(s.childAddress)
		break
	}

	err = s.gatewayPlugin.Start(hndlr)

	return err
}

// Create a new Membrane server
func New(options *MembraneOptions) (*Membrane, error) {
	var childTimeout = 5
	if options.ChildTimeoutSeconds > 0 {
		childTimeout = options.ChildTimeoutSeconds
	}

	var childAddress = "localhost:8080"
	if options.ChildAddress != "" {
		childAddress = options.ChildAddress
	}

	if options.GatewayPlugin == nil {
		return nil, fmt.Errorf("Missing gateway plugin, Gateway plugin must not be nil")
	}

	if !options.TolerateMissingServices {
		if options.EventingPlugin == nil || options.StoragePlugin == nil || options.DocumentsPlugin == nil || options.QueuePlugin == nil || options.AuthPlugin == nil {
			return nil, fmt.Errorf("Missing membrane plugins, if you meant to load with missing plugins set options.TolerateMissingServices to true")
		}
	}

	return &Membrane{
		serviceAddress:          options.ServiceAddress,
		childAddress:            childAddress,
		childUrl:                fmt.Sprintf("http://%s", childAddress),
		childCommand:            options.ChildCommand,
		childTimeoutSeconds:     childTimeout,
		authPlugin:              options.AuthPlugin,
		eventPlugin:             options.EventingPlugin,
		storagePlugin:           options.StoragePlugin,
		documentsPlugin:         options.DocumentsPlugin,
		queuePlugin:             options.QueuePlugin,
		gatewayPlugin:           options.GatewayPlugin,
		supressLogs:             options.SuppressLogs,
		tolerateMissingServices: options.TolerateMissingServices,
		mode:                    options.Mode,
	}, nil
}
