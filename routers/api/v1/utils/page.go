// Package utils owns shared API-boundary utilities, following Gitea's
// routers/api/v1/utils package. Domain pagination remains in models/db.
package utils

import (
	"errors"
	"strconv"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/services/context"
)

var errInvalidListOptions = errors.New("invalid API pagination parameters")

// GetListOptions parses the common page and limit query parameters.
// Like Gitea's routers/api/v1/utils/page.go it is the one API-level
// conversion to db.ListOptions; unlike Gitea it returns an error for invalid
// input to preserve BDSPro's existing 400 response contract.
func GetListOptions(ctx *context.APIContext) (db.ListOptions, error) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		return db.ListOptions{}, errInvalidListOptions
	}
	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		return db.ListOptions{}, errInvalidListOptions
	}
	return db.NewListOptions(page, limit, 20, 100), nil
}
