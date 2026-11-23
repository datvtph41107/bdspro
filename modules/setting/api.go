package setting

import (
	"errors"
	"strings"
)

// APISettings chứa giới hạn REST API, không thuộc routers.
type APISettings struct {
	EnableSwagger       bool
	SwaggerURL          string
	MaxResponseItems    int
	DefaultPagingNum    int
	MaxRequestBodyBytes int64
}

var API APISettings

func loadAPIFrom(cfg ConfigProvider, runMode string, server ServerSettings) (APISettings, error) {
	options := struct {
		MaxResponseItems    int
		DefaultPagingNum    int
		MaxRequestBodyBytes int64
	}{
		MaxResponseItems:    50,
		DefaultPagingNum:    20,
		MaxRequestBodyBytes: 1 << 20,
	}
	if err := mapSettings(cfg, "api", &options); err != nil {
		return APISettings{}, err
	}
	if options.MaxResponseItems < 1 || options.MaxResponseItems > 500 {
		return APISettings{}, errors.New("api.MAX_RESPONSE_ITEMS must be between 1 and 500")
	}
	if options.DefaultPagingNum < 1 || options.DefaultPagingNum > options.MaxResponseItems {
		return APISettings{}, errors.New("api.DEFAULT_PAGING_NUM must be between 1 and MAX_RESPONSE_ITEMS")
	}
	if options.MaxRequestBodyBytes < 1024 || options.MaxRequestBodyBytes > 16<<20 {
		return APISettings{}, errors.New("api.MAX_REQUEST_BODY_BYTES must be between 1024 and 16777216")
	}
	result := APISettings{
		EnableSwagger:       runMode == "development",
		MaxResponseItems:    options.MaxResponseItems,
		DefaultPagingNum:    options.DefaultPagingNum,
		MaxRequestBodyBytes: options.MaxRequestBodyBytes,
	}
	if result.EnableSwagger {
		result.SwaggerURL = strings.TrimRight(server.BrowserPublicOrigin, "/") + "/swagger/index.html"
	}
	return result, nil
}
