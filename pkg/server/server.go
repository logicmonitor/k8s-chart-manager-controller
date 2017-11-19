package server

import (
	"fmt"
	"net"

	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server responsible for handling health checks
// requests.
type Server struct {
	*health.Server
}

// New instantiates and returns a Server.
func New() *Server {
	srv := &Server{
		Server: health.NewServer(),
	}

	srv.SetServingStatus(constants.HealthServerServiceName, healthpb.HealthCheckResponse_NOT_SERVING)

	return srv
}

// Run starts the gRPC server.
func (srv *Server) Run() {
	s := grpc.NewServer()
	healthpb.RegisterHealthServer(s, srv)
	reflection.Register(s)

	// Start the gRPC server.
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", "50000"))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s.Serve(lis) // nolint: errcheck
}
