package web

import "testing"

func TestMountedBeforeRoutingDynamicScopeMatch(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"/orgs/{org}", "/orgs/acme", true},
		{"/orgs/{org}", "/orgs/acme/teams/42", true},
		{"/orgs/:org", "/orgs/acme/teams", true},
		{"/orgs/{org}/teams", "/orgs/acme/teams/42", true},
		{"/orgs/{org}", "/orgs", false},
		{"/orgs/{org}", "/orgs/", false},
		{"/orgs/{org}", "/other/acme", false},
		{"/orgs/{org}", "/orgs/acmeX", true},
		{"/orgs/acme", "/orgs/acmeX", false},
		{"/orgs/acme", "/orgs/acme/teams", true},
		{"/orgs/acme", "/orgs/acme-extra/teams", false},
		{"/", "/anywhere", true},
	}
	for _, tc := range cases {
		if got := matchesMountScope(tc.pattern, tc.path); got != tc.want {
			t.Errorf("matchesMountScope(%q, %q)=%t want=%t", tc.pattern, tc.path, got, tc.want)
		}
	}
}
