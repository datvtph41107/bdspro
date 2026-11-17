package utils

import (
	"fmt"
	"strconv"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// ListOptionsError identifies a public query field without reflecting its
// untrusted value into error messages.
type ListOptionsError struct {
	Field   string
	Message string
}

func (e *ListOptionsError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

// GetListOptions converts page/limit from the HTTP boundary into a bounded
// model ListOptions. Invalid values are rejected, never silently clamped.
func GetListOptions(ctx *appctx.APIContext) (db.ListOptions, error) {
	query := ctx.CurrentRequest().URL.Query()
	page := 1
	size := setting.API.DefaultPagingNum
	if raw, found := query["page"]; found {
		if len(raw) != 1 {
			return db.ListOptions{}, &ListOptionsError{Field: "page", Message: "page must be one positive integer"}
		}
		n, err := strconv.Atoi(raw[0])
		if err != nil || n < 1 {
			return db.ListOptions{}, &ListOptionsError{Field: "page", Message: "page must be a positive integer"}
		}
		page = n
	}
	if raw, found := query["limit"]; found {
		if len(raw) != 1 {
			return db.ListOptions{}, &ListOptionsError{Field: "limit", Message: "limit must be one positive integer"}
		}
		n, err := strconv.Atoi(raw[0])
		if err != nil || n < 1 || n > setting.API.MaxResponseItems {
			return db.ListOptions{}, &ListOptionsError{Field: "limit", Message: "limit is outside configured bounds"}
		}
		size = n
	}
	opts := db.NewListOptions(page, size, setting.API.DefaultPagingNum, setting.API.MaxResponseItems)
	if _, _, ok := opts.GetSkipTake32(); !ok {
		return db.ListOptions{}, &ListOptionsError{Field: "page", Message: "page exceeds supported database offset"}
	}
	return opts, nil
}
