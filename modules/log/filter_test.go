package log

import "testing"

func TestComponentFilter(t *testing.T) {
	components, err := parseComponents("http, sql,default")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		raw  string
		want bool
	}{
		{`{"component":"http"}`, true},
		{`{"component":"sql"}`, true},
		{`{"component":"security"}`, false},
		{`{"message":"startup"}`, true},
		{`{"component":""}`, true},
		{`{"component":123}`, true},
		{`invalid json`, true},
	} {
		if got := componentAllowed(components, []byte(tt.raw)); got != tt.want {
			t.Errorf("%s: got %t want %t", tt.raw, got, tt.want)
		}
	}
	all, err := parseComponents(" ")
	if err != nil || !componentAllowed(all, []byte(`{"component":"anything"}`)) {
		t.Fatal("empty filter must preserve all components")
	}
	for _, invalid := range []string{"http,", ",sql", "http,,sql", "*", "HTTP", "1sql", "sql debug"} {
		if _, err := parseComponents(invalid); err == nil {
			t.Errorf("accepted invalid component list %q", invalid)
		}
	}
}
