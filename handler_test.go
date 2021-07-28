package graphqlgin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
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

var contextQuery = &graphql.Field{
	Type: graphql.Int,
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		value := p.Context.Value("value").(int)
		return value, nil
	},
}

var fileObject = graphql.NewObject(graphql.ObjectConfig{
	Name: "FileObject",
	Fields: graphql.Fields{
		"filename": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				fileheader := p.Source.(*multipart.FileHeader)
				return fileheader.Filename, nil
			},
		},
		"size": &graphql.Field{
			Type: graphql.Int,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				fileheader := p.Source.(*multipart.FileHeader)
				return int(fileheader.Size), nil
			},
		},
	},
})

var singleFileMutation = &graphql.Field{
	Args: graphql.FieldConfigArgument{
		"file": &graphql.ArgumentConfig{
			Type: UploadType,
		},
	},
	Type: fileObject,
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return p.Args["file"], nil
	},
}

var multiFileMutation = &graphql.Field{
	Args: graphql.FieldConfigArgument{
		"files": &graphql.ArgumentConfig{
			Type: graphql.NewList(UploadType),
		},
	},
	Type: graphql.NewList(fileObject),
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return p.Args["files"], nil
	},
}

var singleFileWithValueMutation = &graphql.Field{
	Args: graphql.FieldConfigArgument{
		"file": &graphql.ArgumentConfig{
			Type: UploadType,
		},
		"value": &graphql.ArgumentConfig{
			Type: graphql.Int,
		},
	},
	Type: graphql.NewObject(graphql.ObjectConfig{
		Name: "FileAndValue",
		Fields: graphql.Fields{
			"file": &graphql.Field{
				Type: fileObject,
			},
			"value": &graphql.Field{
				Type: graphql.Int,
			},
		},
	}),
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return p.Args, nil
	},
}

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query: graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"hello":      helloQuery,
			"double":     doubleQuery,
			"ginContext": ginContextQuery,
			"context":    contextQuery,
		},
	}),
	Mutation: graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"singleUpload":       singleFileMutation,
			"multiUpload":        multiFileMutation,
			"singleFileAndValue": singleFileWithValueMutation,
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
	fileType, ok := app.Schema.TypeMap()["Upload"]
	if !ok {
		t.Errorf("File is not found in TypeMap")
	}
	if fileType.Name() != "Upload" {
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

func TestContextFunctionPOST(t *testing.T) {
	app := New(schema, func(c *gin.Context, ctx context.Context) context.Context {
		return context.WithValue(ctx, "value", 5)
	})
	router := setupRouter(app)

	type ctxData struct {
		Value int `json:"context"`
	}
	type ctxResponse struct {
		Data ctxData `json:"data"`
	}

	query := map[string]interface{}{
		"query":         "query context { context }",
		"operationName": "context",
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
	var ctxRes ctxResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &ctxRes); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if ctxRes.Data.Value != 5 {
		t.Errorf("Response incorrect. Found %d, expected %d", ctxRes.Data.Value, 5)
	}
}

func TestSingleFileUploadPOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)

	type fileData struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	}
	type mutationWrapper struct {
		Mutation fileData `json:"singleUpload"`
	}
	type fileResponse struct {
		Data mutationWrapper `json:"data"`
	}

	operations := map[string]interface{}{
		"query":         `mutation uploadFile ( $file: Upload! ) { singleUpload( file: $file ) { filename size } }`,
		"operationName": "uploadFile",
		"variables": map[string]interface{}{
			"file": nil,
		},
	}
	operationsBody, _ := json.Marshal(operations)

	buff := bytes.NewBuffer(nil)
	form := multipart.NewWriter(buff)
	form.WriteField("operations", string(operationsBody))
	form.WriteField("map", `{"file": ["variables.file"]}`)
	w, _ := form.CreateFormFile("file", "hello.txt")
	w.Write([]byte("Hello, World"))
	form.Close()

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", buff)
	request.Header.Add("Content-Type", form.FormDataContentType())

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var res fileResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &res); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if res.Data.Mutation.Filename != "hello.txt" {
		t.Errorf("File name incorrect. expected %s found %s", "hello.txt", res.Data.Mutation.Filename)
	}
	if res.Data.Mutation.Size != 12 {
		t.Errorf("File size incorrect. expected %d found %d", 12, res.Data.Mutation.Size)
	}
}

func TestMultipleFileUploadPOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)

	type fileData struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	}
	type mutationWrapper struct {
		Mutation []fileData `json:"multiUpload"`
	}
	type fileResponse struct {
		Data mutationWrapper `json:"data"`
	}

	operations := map[string]interface{}{
		"query":         `mutation uploadFile ( $files: [Upload!]! ) { multiUpload( files: $files ) { filename size } }`,
		"operationName": "uploadFile",
		"variables": map[string]interface{}{
			"files": []interface{}{nil, nil},
		},
	}
	operationsBody, _ := json.Marshal(operations)

	buff := bytes.NewBuffer(nil)
	form := multipart.NewWriter(buff)
	form.WriteField("operations", string(operationsBody))
	form.WriteField("map", `{"0": ["variables.files.0"], "1": ["variables.files.1"]}`)
	w, _ := form.CreateFormFile("0", "hello.txt")
	w.Write([]byte("Hello, World"))
	w2, _ := form.CreateFormFile("1", "bingo.txt")
	w2.Write([]byte("Bingo"))
	form.Close()

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", buff)
	request.Header.Add("Content-Type", form.FormDataContentType())

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var res fileResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &res); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if res.Data.Mutation[0].Filename != "hello.txt" {
		t.Errorf("File name incorrect. expected %s found %s", "hello.txt", res.Data.Mutation[0].Filename)
	}
	if res.Data.Mutation[0].Size != 12 {
		t.Errorf("File size incorrect. expected %d found %d", 12, res.Data.Mutation[0].Size)
	}
	if res.Data.Mutation[1].Filename != "bingo.txt" {
		t.Errorf("File name incorrect. expected %s found %s", "bingo.txt", res.Data.Mutation[1].Filename)
	}
	if res.Data.Mutation[1].Size != 5 {
		t.Errorf("File size incorrect. expected %d found %d", 5, res.Data.Mutation[1].Size)
	}
}

func TestSingleFileAndValuePOST(t *testing.T) {
	app := New(schema)
	router := setupRouter(app)

	type fileData struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	}
	type resData struct {
		File  fileData `json:"file"`
		Value int      `json:"value"`
	}
	type mutationWrapper struct {
		Mutation resData `json:"singleFileAndValue"`
	}
	type fileResponse struct {
		Data mutationWrapper `json:"data"`
	}

	operations := map[string]interface{}{
		"query":         `mutation uploadFile ( $file: Upload!, $value: Int! ) { singleFileAndValue( file: $file, value: $value ) { file { filename size } value } }`,
		"operationName": "uploadFile",
		"variables": map[string]interface{}{
			"file":  nil,
			"value": 10,
		},
	}
	operationsBody, _ := json.Marshal(operations)

	buff := bytes.NewBuffer(nil)
	form := multipart.NewWriter(buff)
	form.WriteField("operations", string(operationsBody))
	form.WriteField("map", `{"file": ["variables.file"]}`)
	w, _ := form.CreateFormFile("file", "hello.txt")
	w.Write([]byte("Hello, World"))
	form.Close()

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/", buff)
	request.Header.Add("Content-Type", form.FormDataContentType())

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Request failed. Code: %d", recorder.Code)
	}
	var res fileResponse
	body := recorder.Body.Bytes()

	// run tests
	if err := json.Unmarshal(body, &res); err != nil {
		t.Errorf("Response unmarshal failed. Err: %v", err)
	}
	if res.Data.Mutation.File.Filename != "hello.txt" {
		t.Errorf("File name incorrect. expected %s found %s", "hello.txt", res.Data.Mutation.File.Filename)
	}
	if res.Data.Mutation.File.Size != 12 {
		t.Errorf("File size incorrect. expected %d found %d", 12, res.Data.Mutation.File.Size)
	}
	if res.Data.Mutation.Value != 10 {
		t.Errorf("Value incorrect. expected %d found %d", 10, res.Data.Mutation.Value)
	}
}

func ExampleGraphQLApp_simple_usage() {
	// Construct graphql schema
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"hello": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
			},
		}),
	})

	// Create graphql app instance
	app := New(schema)

	// Create gin router
	router := gin.Default()

	// Add graphql handler to the router
	router.POST("/graphql", app.Handler())

	// Run app server
	// router.Run()

	// Example capturing of output
	req, _ := http.NewRequest(
		"POST",
		"/graphql",
		bytes.NewBuffer([]byte(
			`{"query": "query Hello { hello }", "operationName": "Hello", "variables": {}}`,
		)))
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	data := w.Body.Bytes()
	fmt.Println(string(data))

	// Output:
	// {"data":{"hello":"world"}}
}

func ExampleGraphQLApp_single_file_upload() {
	//  create your own json representation of the uploaded file
	uploadInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name: "UploadInfo",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					file := p.Source.(*multipart.FileHeader)
					return file.Filename, nil
				},
			},
			"size": &graphql.Field{
				Type: graphql.Int,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					file := p.Source.(*multipart.FileHeader)
					return file.Size, nil
				},
			},
		},
	})
	// Construct graphql schema
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		// just a dummy query, have to include a query with at least one field
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"hello": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"uploadFile": &graphql.Field{
					// use graphqlgin.UploadType like this for file upload scalar
					Args: graphql.FieldConfigArgument{
						"file": &graphql.ArgumentConfig{
							Type: UploadType,
						},
					},
					// use your own type to respond
					Type: uploadInfoType,
					// handle the uploaded file anyway you want
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Args["file"], nil
					},
				},
			},
		}),
	})

	// Create graphql app
	app := New(schema)

	// Create router
	router := gin.Default()

	// Add graphql handler to the router
	router.POST("/graphql", app.Handler())

	// Run graphql server
	// router.Run()

	// Sample output capturing
	buffer := bytes.NewBuffer(nil)
	form := multipart.NewWriter(buffer)
	form.WriteField(
		"operations",
		`{"query": "mutation UploadFile ($file: Upload!) { uploadFile(file: $file) { filename size } }",
                  "operationName": "UploadFile",
                  "variables": { "file": null }
                 }`,
	)
	form.WriteField(
		"map",
		`{"file": ["variables.file"]}`,
	)
	w, _ := form.CreateFormFile("file", "hello.txt")
	w.Write([]byte("Hello, World"))
	form.Close()
	req, _ := http.NewRequest("POST", "/graphql", buffer)
	req.Header.Add("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	body := rec.Body.Bytes()
	fmt.Println(string(body))

	// Output:
	// {"data":{"uploadFile":{"filename":"hello.txt","size":12}}}
}

func ExampleGraphQLApp_multiple_file_upload() {
	//  create your own json representation of the uploaded file
	uploadInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name: "UploadInfo",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					file := p.Source.(*multipart.FileHeader)
					return file.Filename, nil
				},
			},
			"size": &graphql.Field{
				Type: graphql.Int,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					file := p.Source.(*multipart.FileHeader)
					return file.Size, nil
				},
			},
		},
	})
	// Construct graphql schema
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		// just a dummy query, have to include a query with at least one field
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"hello": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"uploadFiles": &graphql.Field{
					// use graphqlgin.UploadType like this for file upload scalar
					Args: graphql.FieldConfigArgument{
						"files": &graphql.ArgumentConfig{
							Type: graphql.NewList(UploadType),
						},
					},
					// use your own type to respond
					Type: graphql.NewList(uploadInfoType),
					// handle the uploaded file anyway you want
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Args["files"], nil
					},
				},
			},
		}),
	})

	// Create graphql app
	app := New(schema)

	// Create router
	router := gin.Default()

	// Add graphql handler to the router
	router.POST("/graphql", app.Handler())

	// Run graphql server
	// router.Run()

	// Sample output capturing
	buffer := bytes.NewBuffer(nil)
	form := multipart.NewWriter(buffer)
	form.WriteField(
		"operations",
		`{"query": "mutation ($files: [Upload!]!) { uploadFiles(files: $files) { filename size } }",
                  "operationName": "",
                  "variables": { "files": [null, null] }
                 }`,
	)
	form.WriteField(
		"map",
		`{"0": ["variables.files.0"], "1": ["variables.files.1"]}`,
	)
	w, _ := form.CreateFormFile("0", "hello.txt")
	w.Write([]byte("Hello, World"))
	w2, _ := form.CreateFormFile("1", "bingo.txt")
	w2.Write([]byte("Bingo"))
	form.Close()
	req, _ := http.NewRequest("POST", "/graphql", buffer)
	req.Header.Add("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	body := rec.Body.Bytes()
	fmt.Println(string(body))

	// Output:
	// {"data":{"uploadFiles":[{"filename":"hello.txt","size":12},{"filename":"bingo.txt","size":5}]}}
}

func ExampleGraphQLApp_add_custom_headers() {
	// Create schema
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"header": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						c := GetGinContext(p.Context)
						c.Header("Custom-Header", "some-header-value")
						return "hello", nil
					},
				},
			},
		}),
	})

	// Create graphql app
	app := New(schema)

	// Create router
	router := gin.Default()

	// Add graphql handler
	router.POST("/graphql", app.Handler())

	// Run server
	// router.Run()

	// Sample output capture
	req, _ := http.NewRequest(
		"POST",
		"/graphql",
		bytes.NewBuffer([]byte(`{"query": "query { header }", "operationName": "", "variables": {}}`)),
	)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	headers := w.Header()
	body := string(w.Body.Bytes())

	fmt.Println("Headers:")
	fmt.Println(headers)
	fmt.Println("Body:")
	fmt.Println(body)

	// Output:
	// Headers:
	// map[Content-Type:[application/json; charset=utf-8] Custom-Header:[some-header-value]]
	// Body:
	// {"data":{"header":"hello"}}
}

func ExampleGraphQLApp_context_usage() {
	// Your example database
	userStore := []map[string]interface{}{
		map[string]interface{}{
			"name":  "John",
			"email": "a@b.c",
		},
		map[string]interface{}{
			"name":  "Kratos",
			"email": "c@b.a",
		},
	}

	// Create schema
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"users": &graphql.Field{
					Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
						Name: "User",
						Fields: graphql.Fields{
							"name": &graphql.Field{
								Type: graphql.String,
							},
							"email": &graphql.Field{
								Type: graphql.String,
							},
						},
					})),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						// retrieve the store/database from the context
						users := p.Context.Value("userStore").([]map[string]interface{})
						// use it as you want
						return users, nil
					},
				},
			},
		}),
	})

	// Create graphql app
	// You can add your context provider here or when attaching the handler to a route
	app := New(schema, func(c *gin.Context, ctx context.Context) context.Context {
		return context.WithValue(ctx, "userStore", userStore)
	})

	// Create router
	router := gin.Default()

	// Add handler
	router.POST("/graphql", app.Handler())
	// You could add your context providers here too
	// router.POST("/graphql", app.Handler(func(c *gin.Context, ctx context.Context) context.Context {
	// 	return context.WithValue(ctx, "userStore", userStore)
	// }))

	// Run server
	// router.Run()

	// Sample output capture
	req, _ := http.NewRequest(
		"POST",
		"/graphql",
		bytes.NewBuffer([]byte(`{"query": "query { users { name email } }", "operationName": "", "variables": {}}`)),
	)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := string(w.Body.Bytes())

	fmt.Println(body)

	// Output:
	// {"data":{"users":[{"email":"a@b.c","name":"John"},{"email":"c@b.a","name":"Kratos"}]}}
}
