package setting

import (
	"errors"
	"strconv"
	"strings"
)

type APISettings struct {
	EnableSwagger       bool
	SwaggerURL          string
	MaxResponseItems    int
	DefaultPagingNum    int
	MaxRequestBodyBytes int64
}

var API APISettings

func loadAPIFrom(cfg ConfigProvider, runMode string, server ServerSettings) (APISettings, error) {
	sec := cfg.Section("api")
	result := APISettings{
		EnableSwagger:       runMode == "development",
		MaxResponseItems:    50,
		DefaultPagingNum:    20,
		MaxRequestBodyBytes: 1 << 20,
	}
	if v := strings.TrimSpace(sec.Key("MAX_RESPONSE_ITEMS").String()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			return APISettings{}, errors.New("api.MAX_RESPONSE_ITEMS must be between 1 and 500")
		}
		result.MaxResponseItems = n
	}
	if v := strings.TrimSpace(sec.Key("DEFAULT_PAGING_NUM").String()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > result.MaxResponseItems {
			return APISettings{}, errors.New("api.DEFAULT_PAGING_NUM must be between 1 and MAX_RESPONSE_ITEMS")
		}
		result.DefaultPagingNum = n
	}
	if v := strings.TrimSpace(sec.Key("MAX_REQUEST_BODY_BYTES").String()); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1024 || n > 16<<20 {
			return APISettings{}, errors.New("api.MAX_REQUEST_BODY_BYTES must be between 1024 and 16777216")
		}
		result.MaxRequestBodyBytes = n
	}
	if result.DefaultPagingNum > result.MaxResponseItems {
		return APISettings{}, errors.New("api.DEFAULT_PAGING_NUM must not exceed MAX_RESPONSE_ITEMS")
	}
	if result.EnableSwagger {
		result.SwaggerURL = strings.TrimRight(server.BrowserPublicOrigin, "/") + "/swagger/index.html"
	}
	return result, nil
}
