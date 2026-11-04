package main

import (
	"strings"
	"testing"
)

func TestExtractRoutesFromR01Functions(t *testing.T) {
	src := `package api
func (s *Server) registerRoutes(root *Router) {
 root.GET("/health", Health)
}
func (s *Server) registerOrganizationRoutes(route *Group) {
 route.GET("", s.organizationHandler.Index)
 route.PATCH("/:id", s.organizationHandler.Update)
 route.POST("/:id/members", middleware.RequireOwner(), s.organizationHandler.AddMember)
}
func (s *Server) otherFunction(route *Group) {
 route.GET("/not-a-route", Wrong)
}`
	got, err := extractRoutes("routes.go", src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /health", "GET /organizations", "PATCH /organizations/:id",
		"POST /organizations/:id/members",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d routes, want %d: %+v", len(got), len(want), got)
	}
	for i, route := range got {
		if key := routeKey(route.Method, route.Path); key != want[i] {
			t.Errorf("route %d = %s, want %s", i, key, want[i])
		}
	}
}

func TestCheckRoutesCatchesDriftAndDuplicates(t *testing.T) {
	var c contract
	c.Routes = []struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		Access    string `json:"access"`
		Condition string `json:"condition,omitempty"`
	}{{Method: "GET", Path: "/health", Access: "public"}}
	if err := checkRoutes([]route{{Method: "GET", Path: "/health"}}, c); err != nil {
		t.Fatal(err)
	}
	if err := checkRoutes([]route{{Method: "POST", Path: "/health"}}, c); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected drift error, got %v", err)
	}
	c.Routes = append(c.Routes, c.Routes[0])
	if err := checkRoutes([]route{{Method: "GET", Path: "/health"}}, c); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestExtractExportedSwaggerRegistration(t *testing.T) {
	src := `package api
func RegisterSwagger(router *Router, environment string) {
  router.GET("/swagger/*filepath", serveSwagger)
}`
	got, err := extractRoutes("swagger.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || routeKey(got[0].Method, got[0].Path) != "GET /swagger/*filepath" {
		t.Fatalf("exported Swagger registration was not discovered: %+v", got)
	}
}

func TestCheckRoutesPreservesHistoricalBaselineWithAdditions(t *testing.T) {
	var baseline contract
	baseline.Repository = "datvtph41107/bdspro"
	baseline.Routes = []struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		Access    string `json:"access"`
		Condition string `json:"condition,omitempty"`
	}{{Method: "GET", Path: "/health", Access: "public"}}

	additions := baseline
	additions.Routes = append(additions.Routes[:0:0], baseline.Routes[0])
	additions.Routes[0].Path = "/teams/:team_id"
	additions.Routes[0].Access = "verified_contact"

	expected := baseline
	expected.Routes = append(expected.Routes, additions.Routes...)
	whole := []route{
		{Method: "GET", Path: "/health"},
		{Method: "GET", Path: "/teams/:team_id"},
	}
	if err := checkRoutes(whole, expected); err != nil {
		t.Fatal(err)
	}
	if err := checkRoutes(whole, baseline); err == nil {
		t.Fatal("new route must fail without reviewed addition")
	}
	if err := checkRoutes(whole[:1], expected); err == nil {
		t.Fatal("missing reviewed route must fail")
	}
}

func TestReviewedRemovalsMustExistAndCannotRepeat(t *testing.T) {
	var base contract
	base.Routes = []struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		Access    string `json:"access"`
		Condition string `json:"condition,omitempty"`
	}{
		{Method: "GET", Path: "/users/legacy", Access: "public"},
		{Method: "GET", Path: "/health", Access: "public"},
	}
	removals := base
	removals.Routes = append(removals.Routes[:0:0], base.Routes[0])
	next, err := applyRemovals(base, removals)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRoutes([]route{{Method: "GET", Path: "/health"}}, next); err != nil {
		t.Fatal(err)
	}
	if err := checkRoutes([]route{{Method: "GET", Path: "/health"}}, base); err == nil {
		t.Fatal("historical contract was modified")
	}
	removals.Routes[0].Path = "/not-in-baseline"
	if _, err := applyRemovals(base, removals); err == nil || !strings.Contains(err.Error(), "not present") {
		t.Fatalf("unknown reviewed removal allowed: %v", err)
	}
	removals.Routes[0].Path = "/users/legacy"
	removals.Routes = append(removals.Routes, removals.Routes[0])
	if _, err := applyRemovals(base, removals); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate reviewed removal allowed: %v", err)
	}
}
