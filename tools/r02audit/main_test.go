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

