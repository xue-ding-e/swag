package main

import (
	"net/http"

	"github.com/xue-ding-e/swag/testdata/generics_arrays/api"
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @host localhost:4000
// @basePath /api
func main() {
	http.HandleFunc("/posts/", api.GetPosts)
	http.HandleFunc("/posts-multi/", api.GetPostMulti)
	http.HandleFunc("/posts-multis/", api.GetPostArray)
	http.ListenAndServe(":8080", nil)
}
