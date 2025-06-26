package platform

import (
	"app/base/utils"
	"context"
	"net"

	kesselv2 "github.com/project-kessel/inventory-api/api/kessel/inventory/v1beta2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockKesselServer struct {
	kesselv2.UnimplementedKesselInventoryServiceServer
}

func (server MockKesselServer) Check(c context.Context, req *kesselv2.CheckRequest) (*kesselv2.CheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Check not implemented")
}

func (server MockKesselServer) CheckForUpdate(c context.Context, req *kesselv2.CheckForUpdateRequest) (
	*kesselv2.CheckForUpdateResponse, error,
) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckForUpdate not implemented")
}

func (server MockKesselServer) StreamedListObjects(req *kesselv2.StreamedListObjectsRequest,
	streamingServer grpc.ServerStreamingServer[kesselv2.StreamedListObjectsResponse],
) error {
	streamingServer.Send(&kesselv2.StreamedListObjectsResponse{
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

	return nil
}

// InitKessel creates a dummy gRPC server that always responds with the same permission no matter the request.
func initKessel() {
	listener, err := net.Listen("tcp", ":9005") // TODO: handle gRPC port thru env variable (?)
	if err != nil {
		utils.LogFatal("err", err, "failed to create listener for gRPC Kessel mock server")
	}

	grpcServer := grpc.NewServer()
	kesselv2.RegisterKesselInventoryServiceServer(grpcServer, &MockKesselServer{})
	err = grpcServer.Serve(listener)
	if err != nil {
		utils.LogFatal("err", err, "failed to serve gRPC Kessel mock server")
	}
}
