package context

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/datvtph41107/bdspro/modules/setting"
)

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

func (ctx *APIContext) SetLinkHeader(total int64, page, pageSize int) {
	if total <= 0 || page < 1 || pageSize < 1 {
		return
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
	if int64(page) < totalPages {
		add(int64(page)+1, "next")
		add(totalPages, "last")
	}
	if page > 1 {
		add(1, "first")
		add(int64(page)-1, "prev")
	}
	if len(links) != 0 {
		ctx.Header("Link", strings.Join(links, ", "))
		ctx.AppendAccessControlExposeHeaders("Link")
	}
}
