package graphql

import (
	"app/base/structures"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

var hosts = map[int]structures.HostDAO{}

var hostSchema = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "host",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"request": &graphql.Field{
				Type: graphql.String,
			},
			"checksum": &graphql.Field{
				Type: graphql.String,
			},
			"updated": &graphql.Field{
				Type: graphql.DateTime,
			},
		},
	},
)

var querySchema = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"host": &graphql.Field{
				Type: hostSchema,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					query, found := p.Args["id"].(int)
					if found {
						return hosts[query], nil
					}
					return nil, nil
				},
			},
		},
	})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query: querySchema,
	},
)

func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		fmt.Printf("wrong result, unexpected errors: %v", result.Errors)
	}
	return result
}

// TODO: tmp create some hosts to play with
func init() {
	addHost(1, "host1", "abc")
	addHost(2, "host2", "def")
	addHost(3, "host3", "gah")
}

func addHost(id int, request, checksum string) {
	hosts[id] = structures.HostDAO{id, request, checksum, time.Now()}
}

func Handler(c *gin.Context) {
	query := c.Query("query")
	result := executeQuery(query, schema)
	c.JSON(http.StatusOK, result)
	return
}

func PlaygroundHandler(c *gin.Context) {
	h := handler.New(&handler.Config{
		Schema: &schema,
		Pretty: true,
		GraphiQL: false,
		Playground: true,
	})
	h.ServeHTTP(c.Writer, c.Request)
}
