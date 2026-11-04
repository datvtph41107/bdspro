package db

import "math"

// Paginator is the query-pagination contract shared by list options.
type Paginator interface {
	GetSkipTake() (skip, take int)
	IsListAll() bool
}

// ListOptions specifies numbered-page pagination.
type ListOptions struct {
	PageSize int
	Page     int
	ListAll  bool
}

var _ Paginator = ListOptions{}

// NewListOptions applies caller-provided validated paging defaults.
func NewListOptions(page, pageSize, defaultSize, maxSize int) ListOptions {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}
	return ListOptions{Page: page, PageSize: pageSize}
}

func (opts ListOptions) GetSkipTake() (skip, take int) {
	return (opts.Page - 1) * opts.PageSize, opts.PageSize
}

func (opts ListOptions) IsListAll() bool {
	return opts.ListAll
}

// GetSkipTake32 validates bounded SQLC limit and offset parameters before
// multiplication. It never returns wrapped or negative offsets to a SQL query.
// Models decide which domain error to return for invalid pagination.
func (opts ListOptions) GetSkipTake32() (skip, take int32, ok bool) {
	if opts.ListAll || opts.Page < 1 || opts.PageSize < 1 || opts.PageSize > math.MaxInt32 ||
		opts.Page-1 > math.MaxInt32/opts.PageSize {
		return 0, 0, false
	}
	return int32((opts.Page - 1) * opts.PageSize), int32(opts.PageSize), true
}

// AbsoluteListOptions specifies offset-based pagination.
type AbsoluteListOptions struct {
	skip int
	take int
}

var _ Paginator = AbsoluteListOptions{}

func NewAbsoluteListOptions(skip, take, defaultSize, maxSize int) AbsoluteListOptions {
	if skip < 0 {
		skip = 0
	}
	if take <= 0 {
		take = defaultSize
	}
	if take > maxSize {
		take = maxSize
	}
	return AbsoluteListOptions{skip: skip, take: take}
}

func (opts AbsoluteListOptions) GetSkipTake() (skip, take int) {
	return opts.skip, opts.take
}

func (opts AbsoluteListOptions) IsListAll() bool {
	return false
}
