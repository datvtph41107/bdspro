package setting

import (
	"fmt"
	"strconv"
	"strings"
)

// API configures the HTTP API and its documentation. Swagger is development
// only, matching routers/api.SwaggerHandler's environment restriction.
var API struct {
	EnableSwagger    bool
	SwaggerURL       string
	MaxResponseItems int
	DefaultPagingNum int
}

func loadAPIFrom(config ConfigProvider) error {
	sec := config.Section("api")
	API.EnableSwagger = Common.RunMode == "development"
	API.SwaggerURL = ""
	API.MaxResponseItems = 50
	API.DefaultPagingNum = 20
	if v := strings.TrimSpace(sec.Key("MAX_RESPONSE_ITEMS").String()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			return fmt.Errorf("api.MAX_RESPONSE_ITEMS must be between 1 and 500")
		}
		API.MaxResponseItems = n
	}
	if v := strings.TrimSpace(sec.Key("DEFAULT_PAGING_NUM").String()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > API.MaxResponseItems {
			return fmt.Errorf("api.DEFAULT_PAGING_NUM must be between 1 and MAX_RESPONSE_ITEMS")
		}
		API.DefaultPagingNum = n
	}
	if API.DefaultPagingNum > API.MaxResponseItems {
		return fmt.Errorf("api.DEFAULT_PAGING_NUM must not exceed MAX_RESPONSE_ITEMS")
	}
	if API.EnableSwagger {
		API.SwaggerURL = strings.TrimRight(Server.BrowserPublicOrigin, "/") + "/swagger/index.html"
	}
	return nil
}
