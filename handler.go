package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
)

// Function to update or modify the context passed down to the resolver functions
type ContextProviderFn func(c *gin.Context, ctx context.Context) context.Context

// Key for setting `*gin.Context` value of the current request to the context
const GinContextKey = "GinContext"

// Returns a `ContextProviderFn` that will add the current `*gin.Context` value
// to the context passed down to resolver functions.
func GinContextProvider(c *gin.Context, ctx context.Context) context.Context {
	return context.WithValue(
		ctx,
		GinContextKey,
		c,
	)
}

// Extracts and returns the current `*gin.Context` value from the context `ctx`.
func GetGinContext(ctx context.Context) *gin.Context {
	ginContext, _ := ctx.Value(GinContextKey).(*gin.Context)
	return ginContext
}

// Basic GraphQL request parameters
type GraphQLRequestParams struct {
	RequestString  string                 `json:"query" form:"query"`
	VariableValues map[string]interface{} `json:"variables" form:"variables"`
	OperationName  string                 `json:"operationName" form:"operationName"`
}

// GraphQL request parameters including file upload maps and operations
type GraphQLRequest struct {
	RequestString  string                 `json:"query" form:"query"`
	VariableValues map[string]interface{} `json:"variables" form:"variables"`
	OperationName  string                 `json:"operationName" form:"operationName"`
	Operations     GraphQLRequestParams   `json:"operations"`
	UploadMaps     map[string][]string    `json:"map"`
}

// GraphQL app structure
type GraphQLApp struct {
	Schema           graphql.Schema
	ContextProviders []ContextProviderFn
}

// GraphQL scalar to represent file upload variable
var FileType = graphql.NewScalar(
	graphql.ScalarConfig{
		Name:        "File",
		Description: "File upload scalar",
		Serialize: func(value interface{}) interface{} {
			// value will be set by resolver, no need to process
			return value
		},
	},
)

// Constructs a new GraphQL app
func New(schema graphql.Schema, contextProviders ...ContextProviderFn) *GraphQLApp {
	contextProviderFns := []ContextProviderFn{GinContextProvider}
	contextProviderFns = append(contextProviderFns, contextProviders...)
	schema.AppendType(FileType)
	return &GraphQLApp{
		Schema:           schema,
		ContextProviders: contextProviderFns,
	}
}

// Factory function to create `gin.HandlerFunc` for the GraphQL application.
//
// contextProviders will be called before running `graphql.Do` to generate/construct
// the context, and this context will be passed down to the resolver by `graphql.Do`
// function. Any context provider added before or with this function will be executed
// sequentially for each request.
func (app *GraphQLApp) Handler(contextProviders ...ContextProviderFn) gin.HandlerFunc {
	// Add any additional context provided passed to the handler factory
	app.ContextProviders = append(app.ContextProviders, contextProviders...)

	return func(c *gin.Context) {
		// collect graphql request parameters
		var graphqlRequest GraphQLRequest
		if err := c.ShouldBind(&graphqlRequest); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		// TODO: parse operations and map if provided

		// create resolver context
		ctx := context.Background()
		for _, provider := range app.ContextProviders {
			ctx = provider(c, ctx)
		}

		// construct graphql params
		params := graphql.Params{
			Schema:         app.Schema,
			RequestString:  graphqlRequest.RequestString,
			OperationName:  graphqlRequest.OperationName,
			VariableValues: graphqlRequest.VariableValues,
			Context:        ctx,
		}

		// process graphql query
		result := graphql.Do(params)

		// respond
		c.JSON(
			http.StatusOK,
			result,
		)
	}
}
