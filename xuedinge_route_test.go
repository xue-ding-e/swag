package swag

import "testing"

func TestXuedingeRouterPaths(t *testing.T) {
	for _, tc := range []struct {
		name, annotation, path, method string
	}{
		{"chinese", "/用户/订单/{编号} [get]", "/用户/订单/{编号}", "GET"},
		{"gva", "/clientUser/login微信公众号 [post]", "/clientUser/login微信公众号", "POST"},
		{"unicode", "/café/δοκιμή/東京/e\u0301 [get]", "/café/δοκιμή/東京/e\u0301", "GET"},
		{"zwj", "/👩\u200d💻 [get]", "/👩\u200d💻", "GET"},
		{"zwnj", "/می\u200cروم [get]", "/می\u200cروم", "GET"},
		{"rtl", "/العربية/עברית [get]", "/العربية/עברית", "GET"},
		{"emoji", "/标记/😀 [get]", "/标记/😀", "GET"},
		{"tilde-at", "/~用户/@姓名 [get]", "/~用户/@姓名", "GET"},
		{"encoded", "/%E7%94%A8%E6%88%B7/%20 [get]", "/%E7%94%A8%E6%88%B7/%20", "GET"},
		{"wildcard", "/文件/*路径 [get]", "/文件/*路径", "GET"},
		{"existing", "/customer({id})/{proxy+}:move$ [post]", "/customer({id})/{proxy+}:move$", "POST"},
		{"suffix-description", "/base/autoLogin [post] 只能开发模式使用", "/base/autoLogin", "POST"},
		{"suffix-period", "/api [get].", "/api", "GET"},
		{"separator", "/用户\t [get]", "/用户", "GET"},
		{"suffix-method-description", "/用户 [get] 对应 [post] 接口", "/用户", "GET"},
		{"suffix-method", "/用户 [get] [post]", "/用户", "GET"},
		{"suffix-other-path", "/用户 [get] /别的 [post]", "/用户", "GET"},
		{"suffix-invalid-method", "/用户 [get] [unknown]", "/用户", "GET"},
		{"suffix-method-list", "/用户 [get] [get,post]", "/用户", "GET"},
		{"suffix-attached-text", "/用户 [get]junk", "/用户", "GET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation := NewOperation(nil)
			if err := operation.ParseRouterComment(tc.annotation, false); err != nil {
				t.Fatal(err)
			}
			if len(operation.RouterProperties) != 1 || operation.RouterProperties[0].Path != tc.path || operation.RouterProperties[0].HTTPMethod != tc.method {
				t.Fatalf("routes = %#v, want path %q", operation.RouterProperties, tc.path)
			}
		})
	}
}

func TestXuedingeRouterRejectsMalformedPaths(t *testing.T) {
	for _, annotation := range []string{
		"/用户 名 [get]", "/用户\t名 [get]", "/用户\n名 [get]",
		"/用户\x00名 [get]", "/用户\u00a0名 [get]", "/用户\u200b名 [get]",
		"/用户\u202e名 [get]", "/用户\u2028名 [get]",
		// Directional controls and unrelated invisible formatting remain invalid.
		"/用户\u200e名 [get]", "/用户\u200f名 [get]", "/用户\ufeff名 [get]",
		"/bad\xff [get]",
		"/bad% [get]", "/bad%2 [get]", "/bad%GG [get]",
		"/用户?名字=值 [get]", "/用户#片段 [get]", "/用户\\名称 [get]",
		"/用户 [get,post]", "/用户 [unknown]",
		"用户 [get]", "/用户 [get", "/用户\n[get]",
	} {
		t.Run(annotation, func(t *testing.T) {
			operation := NewOperation(nil)
			if err := operation.ParseRouterComment(annotation, false); err == nil {
				t.Fatalf("accepted malformed route: %#v", operation.RouterProperties)
			}
			if len(operation.RouterProperties) != 0 {
				t.Fatal("failed parse must not add a route")
			}
		})
	}
}

func TestXuedingeRoleUsesUpstreamExtension(t *testing.T) {
	operation := NewOperation(nil)
	if err := operation.ParseComment(`@x-role ["管理员","user"]`, nil); err != nil {
		t.Fatal(err)
	}
	roles, ok := operation.Extensions["x-role"].([]interface{})
	if !ok || len(roles) != 2 || roles[0] != "管理员" {
		t.Fatalf("roles = %#v", operation.Extensions["x-role"])
	}
	if err := operation.ParseComment(`@x-role [invalid]`, nil); err == nil {
		t.Fatal("invalid JSON must fail")
	}
}
