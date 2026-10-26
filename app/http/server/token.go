package server

import (
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	"github.com/datvtph41107/bdspro/app/http/resources"
)

type refreshTokenRequest = authrequest.TokenBody
type renewAccessTokenResponse = resources.RenewAccessToken
