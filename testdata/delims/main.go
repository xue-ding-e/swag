package main

import (
	"github.com/xue-ding-e/swag"
	"github.com/xue-ding-e/swag/testdata/delims/api"
	_ "github.com/xue-ding-e/swag/testdata/delims/docs"
)

func ReadDoc() string {
	doc, _ := swag.ReadDoc("CustomDelims")
	return doc
}

// @title Swagger Example API
// @version 1.0
// @description Testing custom template delimeters
// @termsOfService http://swagger.io/terms/

func main() {
	api.MyFunc()
}
