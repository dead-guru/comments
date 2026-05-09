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

func TestBulkModerationFlashReportsPartialFailures(t *testing.T) {
	tests := []struct {
		name   string
		total  int
		failed int
		want   string
	}{
		{name: "none selected", total: 0, failed: 0, want: "No comments selected."},
		{name: "all succeed", total: 3, failed: 0, want: "Comment approved."},
		{name: "some fail", total: 3, failed: 1, want: "Some comments could not be updated."},
		{name: "all fail", total: 3, failed: 3, want: "No comments could be updated."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bulkModerationFlash("approved", tt.total, tt.failed); got != tt.want {
				t.Fatalf("bulkModerationFlash = %q, want %q", got, tt.want)
			}
		})
	}
}
