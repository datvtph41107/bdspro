package db

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
