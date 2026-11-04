package user

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/setting"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/go-redis/redis/v8"
)

const googleOAuthCookie = "bdspro_google_state"

type GoogleHandler struct {
	auth         *auth_service.GoogleAuthService
	config       setting.Config
	flow         *auth_service.GoogleOAuthFlow
	secureCookie bool
	phone        *auth_service.PhoneOTP
}

func NewGoogleHandler(
	auth *auth_service.GoogleAuthService, config setting.Config,
	client *redis.Client, httpClient *http.Client, secureCookie bool,
) *GoogleHandler {
	return &GoogleHandler{
		auth: auth, config: config,
		flow: auth_service.NewGoogleOAuthFlow(config, client, httpClient), secureCookie: secureCookie,
	}
}

// SetPhoneBinding enables the Google reauthentication path in environments
// where phone verification has an actual delivery implementation.
func (g *GoogleHandler) SetPhoneBinding(phone auth_service.PhoneOTP) {
	g.phone = &phone
}
