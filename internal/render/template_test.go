package render

import "testing"

func TestTemplate(t *testing.T) {
	got := Template("Hello {{user_name}} at {{company_name}} for {{role}}", "Alice", "Acme", "Engineer")
	want := "Hello Alice at Acme for Engineer"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
