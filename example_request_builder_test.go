package httpx_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/slashdevops/httpx"
)

// ExampleRequestBuilder_simpleGET demonstrates how to create a simple GET request.
func ExampleRequestBuilder_simpleGET() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/users").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.String())

	// Output:
	// GET
	// https://api.example.com/users
}

// ExampleRequestBuilder_postWithJSON demonstrates how to create a POST request with JSON body.
func ExampleRequestBuilder_postWithJSON() {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := User{Name: "John Doe", Email: "john@example.com"}

	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodPOST().
		WithPath("/users").
		WithJSONBody(user).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.Header.Get("Content-Type"))

	// Output:
	// POST
	// application/json
}

// ExampleRequestBuilder_withQueryParams demonstrates how to add query parameters.
func ExampleRequestBuilder_withQueryParams() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/search").
		WithQueryParam("q", "golang").
		WithQueryParam("limit", "10").
		WithQueryParam("offset", "0").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.URL.String())

	// Output:
	// https://api.example.com/search?limit=10&offset=0&q=golang
}

// ExampleRequestBuilder_withBasicAuth demonstrates how to use basic authentication.
func ExampleRequestBuilder_withBasicAuth() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/protected").
		WithBasicAuth("username", "password").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	authHeader := req.Header.Get("Authorization")
	fmt.Println(authHeader[:6]) // Just print "Basic " prefix

	// Output:
	// Basic
}

// ExampleRequestBuilder_withBearerAuth demonstrates how to use bearer token authentication.
func ExampleRequestBuilder_withBearerAuth() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/api/data").
		WithBearerAuth("your-token-here").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	authHeader := req.Header.Get("Authorization")
	fmt.Println(authHeader[:7]) // Just print "Bearer " prefix

	// Output:
	// Bearer
}

// ExampleRequestBuilder_withAcceptHeader demonstrates how to set the Accept header.
func ExampleRequestBuilder_withAcceptHeader() {
	// Using the WithAccept() convenience method
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/api/data").
		WithAccept("application/json").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Header.Get("Accept"))

	// Output:
	// application/json
}

// ExampleRequestBuilder_withMultipleAcceptTypes demonstrates setting multiple Accept types with quality values.
func ExampleRequestBuilder_withMultipleAcceptTypes() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/content").
		WithAccept("application/json, application/xml;q=0.9, */*;q=0.8").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Header.Get("Accept"))

	// Output:
	// application/json, application/xml;q=0.9, */*;q=0.8
}

// ExampleRequestBuilder_complexRequest demonstrates a complex request with multiple options.
func ExampleRequestBuilder_complexRequest() {
	type RequestData struct {
		Action string `json:"action"`
		Count  int    `json:"count"`
	}

	data := RequestData{Action: "update", Count: 5}

	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodPUT().
		WithPath("/resources/123").
		WithQueryParam("force", "true").
		WithHeader("X-Custom-Header", "custom-value").
		WithUserAgent("MyApp/1.0").
		WithBearerAuth("token123").
		WithJSONBody(data).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.Path)
	fmt.Println(req.Header.Get("User-Agent"))
	fmt.Println(req.Header.Get("X-Custom-Header"))
	fmt.Println(req.Header.Get("Content-Type"))

	// Output:
	// PUT
	// /resources/123
	// MyApp/1.0
	// custom-value
	// application/json
}

// ExampleRequestBuilder_withContext demonstrates how to use a context.
func ExampleRequestBuilder_withContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/data").
		WithContext(ctx).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Context() != nil)

	// Output:
	// true
}

// ExampleRequestBuilder_withMethodHEAD demonstrates how to create a HEAD request.
func ExampleRequestBuilder_withMethodHEAD() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodHEAD().
		WithPath("/resource").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.Path)

	// Output:
	// HEAD
	// /resource
}

// ExampleRequestBuilder_withMethodOPTIONS demonstrates how to create an OPTIONS request.
func ExampleRequestBuilder_withMethodOPTIONS() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodOPTIONS().
		WithPath("/api/users").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.Path)

	// Output:
	// OPTIONS
	// /api/users
}

// ExampleRequestBuilder_withMethodTRACE demonstrates how to create a TRACE request.
func ExampleRequestBuilder_withMethodTRACE() {
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodTRACE().
		WithPath("/debug").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.Path)

	// Output:
	// TRACE
	// /debug
}

// ExampleRequestBuilder_withMethodCONNECT demonstrates how to create a CONNECT request.
func ExampleRequestBuilder_withMethodCONNECT() {
	req, err := httpx.NewRequestBuilder("https://proxy.example.com").
		WithMethodCONNECT().
		WithPath("/tunnel").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(req.Method)
	fmt.Println(req.URL.Path)

	// Output:
	// CONNECT
	// /tunnel
}

// ExampleRequestBuilder_fullExample demonstrates a complete end-to-end example with a test server.
func ExampleRequestBuilder_fullExample() {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	req, err := httpx.NewRequestBuilder(server.URL).
		WithMethodGET().
		WithPath("/api/test").
		WithHeader("Accept", "application/json").
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))

	// Output:
	// 200
	// {"message":"success"}
}
