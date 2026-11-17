package context

import (
	"strconv"
	"strings"

	"github.com/datvtph41107/bdspro/modules/util"
)

// FormString uses net/http's FormValue precedence: urlencoded body, query,
// then multipart body. A missing or empty value falls back to its default.
func (b *Base) FormString(key string, defaults ...string) string {
	value := b.CurrentRequest().FormValue(key)
	if value == "" {
		return util.OptionalArg(defaults, "")
	}
	return value
}

func (b *Base) FormTrim(key string) string {
	return strings.TrimSpace(b.FormString(key))
}

func (b *Base) FormStrings(key string) []string {
	request := b.CurrentRequest()
	if request.Form == nil {
		if err := request.ParseMultipartForm(32 << 20); err != nil {
			return nil
		}
	}
	return request.Form[key]
}

// FormInt/FormInt64 mirror Gitea's permissive HTTP convenience accessors.
// Strict API pagination validation is handled at the router boundary.
func (b *Base) FormInt(key string) int {
	value, _ := strconv.Atoi(b.FormString(key))
	return value
}

func (b *Base) FormInt64(key string) int64 {
	value, _ := strconv.ParseInt(b.FormString(key), 10, 64)
	return value
}
