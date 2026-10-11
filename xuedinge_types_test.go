package swag

import (
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/spec"
)

func TestXuedingeChineseRequestResponse(t *testing.T) {
	const source = `package model
type 请求 struct { Name string ` + "`json:\"名称\"`" + ` }
type 响应 struct { Data interface{} ` + "`json:\"数据\"`" + ` }
type 通用响应[T any] struct { Data T ` + "`json:\"数据\"`" + ` }
type Pair[A, B any] struct { First A; Second B }
`
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "chinese.go", source, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	parser := New()
	if err := parser.packages.collectAstFile(fset, "example/model", "chinese.go", file, ParseModels); err != nil {
		t.Fatal(err)
	}
	if _, err := parser.packages.ParseTypes(); err != nil {
		t.Fatal(err)
	}
	resolve := func(schema *spec.Schema) *spec.Schema {
		t.Helper()
		if schema == nil {
			t.Fatal("missing schema")
		}
		if schema.Ref.GetURL() == nil {
			return schema
		}
		name := strings.TrimPrefix(schema.Ref.GetURL().Fragment, "/definitions/")
		definition, ok := parser.swagger.Definitions[name]
		if !ok {
			t.Fatalf("missing definition %q for ref %s", name, schema.Ref.String())
		}
		return &definition
	}
	checkRequest := func(schema *spec.Schema) {
		t.Helper()
		if schema == nil || schema.Ref.GetURL() == nil || schema.Ref.GetURL().Fragment != "/definitions/model.请求" {
			t.Fatalf("expected request reference, got %#v", schema)
		}
		property, ok := resolve(schema).Properties["名称"]
		if !ok || len(property.Type) != 1 || property.Type[0] != STRING {
			t.Fatalf("missing Chinese JSON string property: %#v", property)
		}
	}
	for _, tc := range []struct{ annotation, kind string }{
		{`@Param payload body 请求 true "中文请求"`, "request"},
		{`@Success 200 {object} 响应 "中文响应"`, "object"},
		{`@Success 200 {array} 请求 "中文数组"`, "array"},
		{`@Success 200 {object} 响应{数据=请求} "中文组合"`, "combined"},
		{`@Success 200 {object} 通用响应[请求] "中文泛型"`, "generic"},
		{`@Success 200 {object} 通用响应[请求]{数据=请求} "中文泛型组合"`, "combined"},
		{"@Param payload body Pair[string,\tint] true \"泛型空白\"", "generic-whitespace"},
	} {
		t.Run(tc.annotation, func(t *testing.T) {
			operation := NewOperation(parser)
			if err := operation.ParseComment(tc.annotation, file); err != nil {
				t.Fatal(err)
			}
			if tc.kind == "request" || tc.kind == "generic-whitespace" {
				if len(operation.Parameters) != 1 || operation.Parameters[0].Schema == nil {
					t.Fatal("missing body parameter schema")
				}
				if tc.kind == "request" {
					checkRequest(operation.Parameters[0].Schema)
				} else if len(resolve(operation.Parameters[0].Schema).Properties) != 2 {
					t.Fatal("generic parameter must preserve both fields")
				}
				return
			}
			schema := operation.Responses.StatusCodeResponses[200].Schema
			switch tc.kind {
			case "object":
				if _, ok := resolve(schema).Properties["数据"]; !ok {
					t.Fatal("missing Chinese response JSON property")
				}
			case "array":
				if schema == nil || len(schema.Type) != 1 || schema.Type[0] != ARRAY || schema.Items == nil {
					t.Fatal("missing response array/items")
				}
				checkRequest(schema.Items.Schema)
			case "combined":
				if schema == nil || len(schema.AllOf) != 2 {
					t.Fatal("missing composed response schema")
				}
				if _, ok := resolve(&schema.AllOf[0]).Properties["数据"]; !ok {
					t.Fatal("missing base response property")
				}
				property := schema.AllOf[1].Properties["数据"]
				checkRequest(&property)
			case "generic":
				property := resolve(schema).Properties["数据"]
				checkRequest(&property)
			}
		})
	}
}
