package graphqlgin

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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
	GraphQLRequestParams
	OperationsString string `json:"-" form:"operations"`
	MapString        string `json:"-" form:"map"`
}

// GraphQL app structure
type GraphQLApp struct {
	Schema           graphql.Schema
	ContextProviders []ContextProviderFn
}

// GraphQL scalar to represent file upload variable
var UploadType = graphql.NewScalar(
	graphql.ScalarConfig{
		Name:        "Upload",
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
	schema.AppendType(UploadType)
	return &GraphQLApp{
		Schema:           schema,
		ContextProviders: contextProviderFns,
	}
}

// Sets leaf object value v in the map m represented by path string.
func set(v interface{}, m interface{}, path string) error {
	var parts []interface{}
	for _, p := range strings.Split(path, ".") {
		if isNumber, err := regexp.MatchString(`\d+`, p); err != nil {
			return err
		} else if isNumber {
			index, _ := strconv.Atoi(p)
			parts = append(parts, index)
		} else {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("empty path found")
	}
	if len(parts) >= 1 && parts[0] != "variables" {
		return fmt.Errorf("first part of path is supposed to be variables")
	}
	// skip the first part as it is supposed to be variables
	for i, p := range parts[1:] {
		last := i+2 == len(parts)
		switch idx := p.(type) {
		case string:
			if last {
				m.(map[string]interface{})[idx] = v
			} else {
				m = m.(map[string]interface{})[idx]
			}
		case int:
			if last {
				m.([]interface{})[idx] = v
			} else {
				m = m.([]interface{})[idx]
			}
		}
	}
	return nil
}

// Shorthand function to construct a graphql error reply
func graphqlErrorReply(message string, err error) map[string]interface{} {
	return map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": fmt.Sprintf(
					"%s (%s)",
					message,
					err,
				),
			},
		},
	}
}

// Factory function to create `gin.HandlerFunc` for the GraphQL application.
//
// Each `contextProviders` will be called before running `graphql.Do` to generate/construct
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
		if len(graphqlRequest.MapString) > 0 && len(graphqlRequest.OperationsString) > 0 {
			// unmarshal graphql operations
			var graphqlOperations GraphQLRequestParams
			if err := json.Unmarshal([]byte(graphqlRequest.OperationsString), &graphqlOperations); err != nil {
				// Reply with an error
				c.JSON(
					http.StatusOK,
					graphqlErrorReply("invalid operations string", err),
				)
				return
			}

			// unmarshal upload/variable map
			variableMap := map[string][]string{}
			if err := json.Unmarshal([]byte(graphqlRequest.MapString), &variableMap); err != nil {
				// Reply with an error
				c.JSON(
					http.StatusOK,
					graphqlErrorReply("invalid map string", err),
				)
				return
			}

			// collect form data from variable map
			uploads := map[*multipart.FileHeader][]string{}
			variables := map[string][]string{}
			for key, path := range variableMap {
				if value, ok := c.GetPostForm(key); ok {
					// this is a plain variable, not a file upload
					variables[value] = path
				} else if fileHeader, err := c.FormFile(key); err != nil {
					// file upload error
					c.JSON(
						http.StatusOK,
						graphqlErrorReply("invalid file upload", err),
					)
					return
				} else if fileHeader != nil {
					// we found a file upload, collect the header
					uploads[fileHeader] = path
				}
			}

			// update graphql request data
			graphqlRequest.RequestString = graphqlOperations.RequestString
			graphqlRequest.OperationName = graphqlOperations.OperationName
			graphqlRequest.VariableValues = graphqlOperations.VariableValues

			// set found form values to request variable values
			for value, paths := range variables {
				for _, path := range paths {
					if err := set(value, graphqlRequest.VariableValues, path); err != nil {
						c.JSON(
							http.StatusOK,
							graphqlErrorReply("could not set variable", err),
						)
						return
					}
				}
			}

			// set found form file uploads to request variable values
			for file, paths := range uploads {
				for _, path := range paths {
					if err := set(file, graphqlRequest.VariableValues, path); err != nil {
						c.JSON(
							http.StatusOK,
							graphqlErrorReply("could not set variable", err),
						)
						return
					}
				}
			}
		}

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
