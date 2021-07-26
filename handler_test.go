package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
)

var helloQuery = &graphql.Field{
	Type: graphql.String,
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return "world", nil
	},
}

var doubleQuery = &graphql.Field{
	Type: graphql.Int,
	Args: graphql.FieldConfigArgument{
		"value": &graphql.ArgumentConfig{
			Type: graphql.Int,
		},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		value, _ := p.Args["value"].(int)
		return value * 2, nil
	},
}

var ginContextQuery = &graphql.Field{
	Type: graphql.Boolean,
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		_, ok := p.Context.Value(GinContextKey).(*gin.Context)
		return ok, nil
	},
}

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query: graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"hello":      helloQuery,
			"double":     doubleQuery,
			"ginContext": ginContextQuery,
		},
	}),
})

func setupRouter(app *GraphQLApp, contextProviders ...ContextProviderFn) *gin.Engine {
	router := gin.Default()
	router.POST("/", app.Handler(contextProviders...))
	router.GET("/", app.Handler(contextProviders...))
	return router
}

func TestSimplePOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)
	type helloData struct {
		Hello string `json:"hello"`
	}
	type helloResponse struct {
		Data helloData `json:"data"`
	}

	query := map[string]interface{}{
		"query":         "query hello { hello }",
		"operationName": "hello",
		"variables":     map[string]interface{}{},
	}
	queryBody, _ := json.Marshal(query)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", bytes.NewBuffer(queryBody))
	request.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var helloRes helloResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &helloRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if helloRes.Data.Hello != "world" {
		t.Errorf("Response incorrect. Found %s, expected %s", helloRes.Data.Hello, "world")
	}
}

func TestSimpleGET(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)
	type helloData struct {
		Hello string `json:"hello"`
	}
	type helloResponse struct {
		Data helloData `json:"data"`
	}

	query := url.Values{
		"query":         []string{"query hello{hello}"},
		"operationName": []string{"hello"},
		"variables":     []string{"{}"},
	}
	queryParams := query.Encode()

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/?"+queryParams, nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var helloRes helloResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &helloRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if helloRes.Data.Hello != "world" {
		t.Errorf("Response incorrect. Found %s, expected %s", helloRes.Data.Hello, "world")
	}
}

func TestValiablesPOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)
	type doubleData struct {
		Double int64 `json:"double"`
	}
	type doubleResponse struct {
		Data doubleData `json:"data"`
	}

	query := map[string]interface{}{
		"query":         "query double ($value: Int) { double(value: $value) }",
		"operationName": "double",
		"variables": map[string]interface{}{
			"value": 5,
		},
	}
	queryBody, _ := json.Marshal(query)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", bytes.NewBuffer(queryBody))
	request.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var doubleRes doubleResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &doubleRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if doubleRes.Data.Double != 10 {
		t.Errorf("Response incorrect. Found %v, expected %v", doubleRes.Data.Double, 10)
	}
}

func TestValiablesGET(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)
	type doubleData struct {
		Double int64 `json:"double"`
	}
	type doubleResponse struct {
		Data doubleData `json:"data"`
	}

	query := url.Values{
		"query":         []string{"query double($value:Int){double(value:$value)}"},
		"operationName": []string{"double"},
		"variables":     []string{"{\"value\":5}"},
	}
	queryParams := query.Encode()

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/?"+queryParams, nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var doubleRes doubleResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &doubleRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if doubleRes.Data.Double != 10 {
		t.Errorf("Response incorrect. Found %v, expected %v", doubleRes.Data.Double, 10)
	}
}

func TestFileTypeScalarAdded(t *testing.T) {
	app := New(schema)
	fileType, ok := app.Schema.TypeMap()["File"]
	if !ok {
		t.Errorf("File is not found in TypeMap")
	}
	if fileType.Name() != "File" {
		t.Errorf("File is not found in TypeMap")
	}
}

func TestGinContextPOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)
	type gcData struct {
		Found bool `json:"ginContext"`
	}
	type gcResponse struct {
		Data gcData `json:"data"`
	}

	query := map[string]interface{}{
		"query":         "query checkGinContext { ginContext }",
		"operationName": "checkGinContext",
		"variables":     map[string]interface{}{},
	}
	queryBody, _ := json.Marshal(query)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", bytes.NewBuffer(queryBody))
	request.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var helloRes gcResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &helloRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if !helloRes.Data.Found {
		t.Errorf("Response incorrect. Found %v, expected %v", helloRes.Data.Found, true)
	}
}
