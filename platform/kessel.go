package platform

import (
	"app/base/utils"
	"context"
	"net"

	kesselv2 "github.com/project-kessel/inventory-api/api/kessel/inventory/v1beta2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ListObjectStreamingServer = grpc.ServerStreamingServer[kesselv2.StreamedListObjectsResponse]

type MockKesselServer struct {
	kesselv2.UnimplementedKesselInventoryServiceServer
}

func (server MockKesselServer) Check(_ context.Context, _ *kesselv2.CheckRequest) (*kesselv2.CheckResponse, error) {
	return &kesselv2.CheckResponse{Allowed: kesselv2.Allowed_ALLOWED_TRUE}, nil
}

func (server MockKesselServer) CheckForUpdate(_ context.Context, _ *kesselv2.CheckForUpdateRequest) (
	*kesselv2.CheckForUpdateResponse, error,
) {
	return &kesselv2.CheckForUpdateResponse{Allowed: kesselv2.Allowed_ALLOWED_TRUE}, nil
}

func (server MockKesselServer) StreamedListObjects(_ *kesselv2.StreamedListObjectsRequest,
	streamingServer ListObjectStreamingServer,
) error {
	return streamingServer.Send(&kesselv2.StreamedListObjectsResponse{
		Object: &kesselv2.ResourceReference{
			ResourceType: "workspace",
			ResourceId:   "inventory-group-1",
			// Reporter: &kesselv2.ReporterReference{
			// 	Type:       "rbac",
			// 	InstanceId: new(string),
			// },
		},
		// Pagination:       &kesselv2.ResponsePagination{ContinuationToken: ""},
		// ConsistencyToken: &kesselv2.ConsistencyToken{Token: ""},
	})
}

// InitKessel creates a dummy gRPC server that always responds with the same permission no matter the request.
func initKessel() {
	listener, err := net.Listen("tcp", ":9005") // #nosec G102 (ignore gosec warning: Binds to all network interfaces)
	if err != nil {
		utils.LogFatal("err", err, "failed to create listener for gRPC Kessel mock server")
	}

	grpcServer := grpc.NewServer()
	kesselv2.RegisterKesselInventoryServiceServer(grpcServer, &MockKesselServer{})
	reflection.Register(grpcServer)
	err = grpcServer.Serve(listener)
	if err != nil {
		utils.LogFatal("err", err, "failed to serve gRPC Kessel mock server")
	}
}
