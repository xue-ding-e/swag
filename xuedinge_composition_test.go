package swag

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/spec"
)

func chineseCompositionFixture(t *testing.T) (*Parser, *ast.File) {
	t.Helper()
	source := `package model
 type Embedded struct { Name string ` + "`json:\"名称\"`" + ` }
 type 请求 struct { Embedded }
 type User用户 struct { Embedded }
 type Alias struct { Embedded } // @name 响应👩‍💻
 type 响应[A, B any] struct { First A; Second B }
 type Pair[A, B any] struct { First A; Second B }
 type 容器[T any] struct { Data T ` + "`json:\"数据\"`" + ` }
 `
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "composition.go", source, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	parser := New()
	if err := parser.packages.collectAstFile(fset, "example/model", "composition.go", file, ParseModels); err != nil {
		t.Fatal(err)
	}
	if _, err := parser.packages.ParseTypes(); err != nil {
		t.Fatal(err)
	}
	return parser, file
}

func resolveComposition(t *testing.T, parser *Parser, schema *spec.Schema) *spec.Schema {
	t.Helper()
	if schema == nil {
		t.Fatal("missing schema")
	}
	if schema.Ref.GetURL() == nil {
		return schema
	}
	key := strings.TrimPrefix(schema.Ref.GetURL().Fragment, "/definitions/")
	found, ok := parser.swagger.Definitions[key]
	if !ok {
		t.Fatalf("missing reference %q", key)
	}
	return &found
}

func TestXuedingeNestedGenericComposition(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	for _, expression := range []string{
		`响应[请求,string]{First=Pair[请求,[]请求],Second=[]model.请求}`,
		`响应[请求, string]{First=Pair[请求, []请求],Second=[]model.请求}`,
		`容器[Pair[请求,string]]{First=Pair[请求,[]请求],Second=[]model.请求}`,
		`object{First=Pair[请求,[]请求],Second=[]model.请求}`,
	} {
		t.Run(expression, func(t *testing.T) {
			operation := NewOperation(parser)
			if err := operation.ParseComment(`@Success 200 {object} `+expression+` "嵌套组合"`, file); err != nil {
				t.Fatal(err)
			}
			schema := operation.Responses.StatusCodeResponses[200].Schema
			if len(schema.AllOf) == 2 {
				schema = &schema.AllOf[1]
			}
			first, ok := schema.Properties["First"]
			if !ok {
				t.Fatal("missing first override")
			}
			pair := resolveComposition(t, parser, &first)
			request := pair.Properties["first"]
			if len(pair.Properties) == 0 {
				t.Fatal("missing generic pair fields")
			}
			if _, ok := pair.Properties["first"]; !ok {
				request = pair.Properties["First"]
			}
			requestSchema := resolveComposition(t, parser, &request)
			if p, ok := requestSchema.Properties["名称"]; !ok || len(p.Type) != 1 || p.Type[0] != STRING {
				t.Fatal("anonymous embedded Chinese property missing")
			}
			pairSecond, ok := pair.Properties["second"]
			if !ok {
				pairSecond = pair.Properties["Second"]
			}
			if len(pairSecond.Type) != 1 || pairSecond.Type[0] != ARRAY || pairSecond.Items == nil {
				t.Fatal("second generic argument is not an array")
			}
			if _, ok := resolveComposition(t, parser, pairSecond.Items.Schema).Properties["名称"]; !ok {
				t.Fatal("second generic argument lost request items")
			}
			second, ok := schema.Properties["Second"]
			if !ok || len(second.Type) != 1 || second.Type[0] != ARRAY || second.Items == nil {
				t.Fatal("missing second array override")
			}
			if p, ok := resolveComposition(t, parser, second.Items.Schema).Properties["名称"]; !ok || len(p.Type) != 1 || p.Type[0] != STRING {
				t.Fatal("qualified Chinese array reference incorrect")
			}
		})
	}
}

func TestXuedingeNestedCompositionContainers(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	for _, annotation := range []string{
		`@Success 200 {object} object{数据=object{配对=Pair[请求,[]请求],列表=[]请求},映射=map[string]Pair[请求,string]} "嵌套"`,
		`@Param payload body object{数据=object{配对=Pair[请求,[]请求],列表=[]请求},映射=map[string]Pair[请求,string]} true "请求"`,
	} {
		operation := NewOperation(parser)
		if err := operation.ParseComment(annotation, file); err != nil {
			t.Fatal(err)
		}
		var schema *spec.Schema
		if len(operation.Parameters) > 0 {
			schema = operation.Parameters[0].Schema
		} else {
			schema = operation.Responses.StatusCodeResponses[200].Schema
		}
		data := schema.Properties["数据"]
		pair := data.Properties["配对"]
		if len(resolveComposition(t, parser, &pair).Properties) != 2 {
			t.Fatal("nested generic lost fields")
		}
		list := data.Properties["列表"]
		if list.Items == nil || len(resolveComposition(t, parser, list.Items.Schema).Properties) != 1 {
			t.Fatal("nested array lost embedded field")
		}
		mapping := schema.Properties["映射"]
		if mapping.AdditionalProperties == nil || mapping.AdditionalProperties.Schema == nil || len(resolveComposition(t, parser, mapping.AdditionalProperties.Schema).Properties) != 2 {
			t.Fatal("generic map lost fields")
		}
		pairSchema := resolveComposition(t, parser, mapping.AdditionalProperties.Schema)
		first, ok := pairSchema.Properties["first"]
		if !ok {
			first = pairSchema.Properties["First"]
		}
		if _, ok := resolveComposition(t, parser, &first).Properties["名称"]; !ok {
			t.Fatal("generic map first argument lost reference")
		}
		second, ok := pairSchema.Properties["second"]
		if !ok {
			second = pairSchema.Properties["Second"]
		}
		if len(second.Type) != 1 || second.Type[0] != STRING {
			t.Fatal("generic map second argument lost string type")
		}

	}
}

func TestXuedingeCompositionRejectsMalformedNesting(t *testing.T) {
	for _, expression := range []string{
		`object{数据=Pair[string,int}`, `object{数据=Pair[string,int]]}`, `object{数据=object{值=string]}}`,
		`object{数据=string}junk`, `object{数据=string}{别的=int}`, `Pair[string,int]{数据=string`,
		strings.Repeat("[]", 1024) + "string",
	} {
		t.Run(expression[:min(len(expression), 60)], func(t *testing.T) {
			if _, err := parseObjectSchema(nil, expression, nil); err == nil {
				t.Fatal("accepted malformed or excessive nesting")
			}
		})
	}
}

func TestXuedingeQualifiedUnicodeTypes(t *testing.T) {
	parser, _ := chineseCompositionFixture(t)
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "api.go", `package api; import 中文 "example/model"`, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := parser.packages.collectAstFile(fset, "example/api", "api.go", file, ParseModels); err != nil {
		t.Fatal(err)
	}
	for _, annotation := range []string{
		`@Param 请求 body 中文.User用户 true "中文请求"`,
		`@Success 200 {object} 中文.Pair[中文.User用户,[]中文.User用户]{数据=中文.User用户} "中文响应"`,
	} {
		operation := NewOperation(parser)
		if err := operation.ParseComment(annotation, file); err != nil {
			t.Fatal(err)
		}
		var schema *spec.Schema
		if len(operation.Parameters) > 0 {
			schema = operation.Parameters[0].Schema
		} else {
			response := operation.Responses.StatusCodeResponses[200].Schema
			if response == nil || len(response.AllOf) != 2 {
				t.Fatal("missing qualified combination")
			}
			property := response.AllOf[1].Properties["数据"]
			schema = &property
		}
		if p, ok := resolveComposition(t, parser, schema).Properties["名称"]; !ok || len(p.Type) != 1 || p.Type[0] != STRING {
			t.Fatal("qualified Unicode field lost")
		}
	}
}

func TestXuedingeGenericSyntaxBoundaries(t *testing.T) {
	for _, invalid := range []string{"", " ", "Pair[]", "Pair[,string]", "Pair[string,,int]", "Pair[string,int]]", "Pair[string][int]", "Pair[[]string,int"} {
		if name, _ := splitGenericsTypeName(invalid); name != "" {
			t.Errorf("accepted %q", invalid)
		}
	}
	for _, valid := range []string{"Pair[string,int]", "Pair[string,int,]", "Pair[请求, []请求]", "Pair[map[string]Pair[请求,int],[]请求]"} {
		name, args := splitGenericsTypeName(valid)
		if name != "Pair" || len(args) != 2 {
			t.Errorf("lost generic arguments in %q: %q %v", valid, name, args)
		}
	}
}

func TestXuedingeTypeSafetyBoundaries(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	for _, expression := range []string{
		"请求\u200d", "请求\u0301", "1请求", "请求\x00", "请求\u202e", string([]byte{0xff}),
		"Pair[" + strings.Repeat("[]", 1024) + "请求,string]",
		"object{" + strings.Repeat("a", 65537) + "=string}",
	} {
		if _, err := parseObjectSchema(parser, expression, file); err == nil {
			t.Errorf("accepted invalid type %.60q", expression)
		}
	}
	// Recursive Go models are handled separately; finite annotation containers are bounded.
	if _, err := parseObjectSchema(nil, strings.Repeat("[]", 32)+"string", nil); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkXuedingeNestedTypeScan(b *testing.B) {
	expression := "object{" + strings.Repeat("数据=Pair[请求,[]请求],", 256) + "末项=string}"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := typeDelimiterIndices(expression, '{'); err != nil {
			b.Fatal(err)
		}
	}
}

func TestXuedingeCompositionUnicodeJSONKeys(t *testing.T) {
	schema, err := parseObjectSchema(nil, "object{👩‍💻=string,می‌روم=int}", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"👩‍💻", "می‌روم"} {
		if _, ok := schema.Properties[key]; !ok {
			t.Fatalf("Unicode JSON key changed: %q", key)
		}
	}
	if name, _ := splitGenericsTypeName("Pair[\xff,int]"); name != "" {
		t.Fatal("accepted invalid UTF-8 generic")
	}
}

func TestXuedingeGenericContainerBudget(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	expression := "请求"
	for i := 0; i < 3; i++ {
		expression = "容器[" + strings.Repeat("[]", 64) + expression + "]"
	}
	if _, err := parseObjectSchema(parser, expression, file); err == nil {
		t.Fatal("nested generic layers must share annotation container budget")
	}
	// Width does not increase nesting depth.
	expression = "object{" + strings.Repeat("项=Pair[请求,[]请求],", 256) + "尾=string}"
	if _, err := parseObjectSchema(parser, expression, file); err != nil {
		t.Fatal(err)
	}
}

func TestXuedingeAnnotationBudgetBoundaries(t *testing.T) {
	if _, err := typeDelimiterIndices(strings.Repeat("a", maxSchemaTypeLength), ','); err != nil {
		t.Fatal(err)
	}
	if _, err := typeDelimiterIndices(strings.Repeat("a", maxSchemaTypeLength+1), ','); err == nil {
		t.Fatal("length budget not enforced")
	}
	if _, err := parseObjectSchema(nil, strings.Repeat("[]", maxSchemaTypeDepth)+"string", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := parseObjectSchema(nil, strings.Repeat("[]", maxSchemaTypeDepth+1)+"string", nil); err == nil {
		t.Fatal("container budget not enforced")
	}
}

func TestXuedingeUnicodeSchemaAlias(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	operation := NewOperation(parser)
	if err := operation.ParseComment(`@Success 200 {object} 响应👩‍💻 "别名"`, file); err != nil {
		t.Fatal(err)
	}
	schema := operation.Responses.StatusCodeResponses[200].Schema
	if schema == nil || schema.Ref.GetURL() == nil || schema.Ref.GetURL().Fragment != "/definitions/响应👩‍💻" {
		t.Fatal("schema alias reference changed")
	}
	if _, ok := resolveComposition(t, parser, schema).Properties["名称"]; !ok {
		t.Fatal("alias schema lost embedded property")
	}
}

func TestXuedingeNestedMapKeyPreservesValue(t *testing.T) {
	parser, file := chineseCompositionFixture(t)
	for _, expression := range []string{"map[Pair[string,int]]请求", "容器[map[Pair[string,int]]请求]", "容器[map[[2]string]请求]"} {
		schema, err := parseObjectSchema(parser, expression, file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(expression, "容器") {
			property := resolveComposition(t, parser, schema).Properties["数据"]
			schema = &property
		}
		schema = resolveComposition(t, parser, schema)
		if schema.AdditionalProperties == nil || schema.AdditionalProperties.Schema == nil {
			t.Fatalf("map lost its value schema for %s", expression)
		}
		if _, ok := resolveComposition(t, parser, schema.AdditionalProperties.Schema).Properties["名称"]; !ok {
			t.Fatal("nested key truncated the map value")
		}
	}
}
