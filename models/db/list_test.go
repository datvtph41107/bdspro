package db

import "testing"

// TestPaginator follows Gitea's shared Paginator contract test pattern.
func TestPaginator(t *testing.T) {
	const defaultSize, maxSize = 20, 100
	tests := []struct {
		name      string
		paginator Paginator
		skip      int
		take      int
	}{
		{"defaults", NewListOptions(-1, -1, defaultSize, maxSize), 0, 20},
		{"page two", NewListOptions(2, 10, defaultSize, maxSize), 10, 10},
		{"maximum limit", NewListOptions(3, 500, defaultSize, maxSize), 200, 100},
		{"absolute defaults", NewAbsoluteListOptions(-1, -1, defaultSize, maxSize), 0, 20},
		{"absolute offset", NewAbsoluteListOptions(2, 10, defaultSize, maxSize), 2, 10},
		{"absolute maximum", NewAbsoluteListOptions(2, 500, defaultSize, maxSize), 2, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, take := tt.paginator.GetSkipTake()
			if skip != tt.skip || take != tt.take {
				t.Fatalf("GetSkipTake = (%d, %d), want (%d, %d)", skip, take, tt.skip, tt.take)
			}
		})
	}
}

func TestListAllContract(t *testing.T) {
	opts := ListOptions{Page: 1, PageSize: 20, ListAll: true}
	if !opts.IsListAll() {
		t.Fatal("ListAll option was lost")
	}
	if NewAbsoluteListOptions(0, 10, 20, 100).IsListAll() {
		t.Fatal("absolute pagination must not enable ListAll")
	}
}

func TestListOptionsSQLPagination(t *testing.T) {
	tests := []struct {
		name       string
		opts       ListOptions
		skip, take int32
		valid      bool
	}{
		{"first page", ListOptions{Page: 1, PageSize: 20}, 0, 20, true},
		{"second page", ListOptions{Page: 2, PageSize: 20}, 20, 20, true},
		{"largest offset", ListOptions{Page: 2147483647, PageSize: 1}, 2147483646, 1, true},
		{"overflow", ListOptions{Page: 2147483647, PageSize: 2}, 0, 0, false},
		{"page zero", ListOptions{Page: 0, PageSize: 20}, 0, 0, false},
		{"negative page", ListOptions{Page: -1, PageSize: 20}, 0, 0, false},
		{"size zero", ListOptions{Page: 1, PageSize: 0}, 0, 0, false},
		{"size out of range", ListOptions{Page: 1, PageSize: 2147483647 + 1}, 0, 0, false},
		{"list all", ListOptions{Page: 1, PageSize: 20, ListAll: true}, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, take, ok := tt.opts.GetSkipTake32()
			if skip != tt.skip || take != tt.take || ok != tt.valid {
				t.Fatalf("GetSkipTake32 = (%d, %d, %v), want (%d, %d, %v)",
					skip, take, ok, tt.skip, tt.take, tt.valid)
			}
		})
	}
}
