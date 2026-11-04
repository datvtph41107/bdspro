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
