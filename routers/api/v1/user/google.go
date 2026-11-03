package user

import (
	"errors"
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/go-redis/redis/v8"
)

const (
	googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL         = "https://oauth2.googleapis.com/token"
	googleOAuthCookie      = "bdspro_google_state"
	googleOAuthTTL         = 5 * time.Minute
)

var errGoogleProviderUnavailable = errors.New("Google OAuth provider unavailable")

type GoogleHandler struct {
	auth         *auth_service.GoogleAuthService
	config       setting.Config
	redis        *redis.Client
	httpClient   *http.Client
	secureCookie bool
	phone        *auth_service.PhoneOTP
}
type googlePending struct {
	Purpose   string `json:"purpose"`
	AccountID int64  `json:"account_id"`
	Verifier  string `json:"verifier"`
	Nonce     string `json:"nonce"`
	Phone     string `json:"phone,omitempty"`
}

func NewGoogleHandler(
	auth *auth_service.GoogleAuthService, config setting.Config,
	client *redis.Client, httpClient *http.Client, secureCookie bool,
) *GoogleHandler {
	return &GoogleHandler{
		auth: auth, config: config,
		redis: client, httpClient: httpClient, secureCookie: secureCookie,
	}
}

// SetPhoneBinding enables the Google reauthentication path in environments
// where phone verification has an actual delivery implementation.
func (g *GoogleHandler) SetPhoneBinding(phone auth_service.PhoneOTP) {
	g.phone = &phone
}
