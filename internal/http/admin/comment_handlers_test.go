package admin

import "testing"

func TestAdminRedirectLocationAllowsOnlyLocalAdminPaths(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		flash string
		want  string
	}{
		{name: "admin root", raw: "/admin", want: "/admin"},
		{name: "admin child", raw: "/admin/comments?status=pending", want: "/admin/comments?status=pending"},
		{name: "adds flash", raw: "/admin/comments?status=pending", flash: "Comment approved.", want: "/admin/comments?flash=Comment+approved.&status=pending"},
		{name: "rejects absolute", raw: "https://evil.example/admin", want: "/admin"},
		{name: "rejects protocol relative", raw: "//evil.example/admin", want: "/admin"},
		{name: "rejects non admin path", raw: "/login", want: "/admin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adminRedirectLocation(tt.raw, tt.flash); got != tt.want {
				t.Fatalf("adminRedirectLocation(%q, %q) = %q, want %q", tt.raw, tt.flash, got, tt.want)
			}
		})
	}
}
