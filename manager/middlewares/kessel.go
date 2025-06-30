package middlewares

import (
	"app/base/utils"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	kesselv2 "github.com/project-kessel/inventory-api/api/kessel/inventory/v1beta2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: do proper logging
// TODO: use secure credentials

func hasPermissionKessel(c *gin.Context) bool {
	// create a connection
	conn, err := grpc.NewClient("platform:9005", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		utils.LogFatal("failed to connect to gRPC server")
	}
	defer conn.Close()

	// create a client and context
	// TODO: maybe use gin.Context instead (?)
	client := kesselv2.NewKesselInventoryServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// make an RPC
	// TODO: obj, rel, subj go into StreamedListObjectsRequest{}
	stream, err := client.StreamedListObjects(ctx, &kesselv2.StreamedListObjectsRequest{})
	if err != nil {
		utils.LogFatal("err", err, "failed to receive gRPC data thru StreamedListObjects")
	}

	// receive response(s)
	done := make(chan bool)
	var finalResponse []*kesselv2.StreamedListObjectsResponse

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				utils.LogFatal("cannot receive gRPC data")
			}
			finalResponse = append(finalResponse, res)
			utils.LogInfo("gRPC response received %v", res)
		}
	}()

	<-done

	// dirty check
	if finalResponse[0].Object.ResourceId == "inventory-group-1" {
		return true
	}

	return false
}

func Kessel() gin.HandlerFunc {
	// if !config.EnableKessel {
	// 	return func(_ *gin.Context) {}
	// }

	return func(c *gin.Context) {
		if hasPermissionKessel(c) {
			utils.LogInfo("KESSEL CHECK SUCCESSFUL")
			return
		}
		utils.LogError("NO KESSEL PERMISSION")
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}

}
