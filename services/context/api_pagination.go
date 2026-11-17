package context

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/datvtph41107/bdspro/modules/setting"
)

// AppendAccessControlExposeHeaders makes API metadata visible to CORS clients
// while retaining other registered exposed headers.
func (b *Base) AppendAccessControlExposeHeaders(names ...string) {
	header := b.source.ResponseHeader()
	existing := header.Get("Access-Control-Expose-Headers")
	seen := make(map[string]bool)
	merged := make([]string, 0, len(names)+2)
	for _, item := range strings.Split(existing, ",") {
		name := strings.TrimSpace(item)
		if name != "" && !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			merged = append(merged, name)
		}
	}
	for _, name := range names {
		if name != "" && !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			merged = append(merged, name)
		}
	}
	header.Set("Access-Control-Expose-Headers", strings.Join(merged, ", "))
}

func (b *Base) SetTotalCountHeader(total int64) {
	b.Header("X-Total-Count", strconv.FormatInt(total, 10))
	b.AppendAccessControlExposeHeaders("X-Total-Count")
}

// SetLinkHeader emits Gitea-style RFC 8288 Link headers for 1-based paging.
// Public links come only from validated server settings (never Host/XFH).
func (ctx *APIContext) SetLinkHeader(total int64, pageSize int) {
	if total <= 0 || pageSize < 1 {
		return
	}
	currentPage := ctx.FormInt("page")
	if currentPage < 1 {
		currentPage = 1
	}
	totalPages := 1 + (total-1)/int64(pageSize)
	links := make([]string, 0, 4)
	add := func(page int64, relation string) {
		target := *ctx.CurrentRequest().URL
		query := target.Query()
		query.Set("page", strconv.FormatInt(page, 10))
		target.RawQuery = query.Encode()
		uri := target.RequestURI()
		if uri == "" {
			return
		}
		if origin := strings.TrimRight(setting.Server.BrowserPublicOrigin, "/"); origin != "" {
			uri = origin + uri
		}
		links = append(links, fmt.Sprintf("<%s>; rel=%q", uri, relation))
	}
	if int64(currentPage) < totalPages {
		add(int64(currentPage)+1, "next")
		add(totalPages, "last")
	}
	if currentPage > 1 {
		add(1, "first")
		add(int64(currentPage)-1, "prev")
	}
	if len(links) != 0 {
		ctx.Header("Link", strings.Join(links, ", "))
		ctx.AppendAccessControlExposeHeaders("Link")
	}
}
