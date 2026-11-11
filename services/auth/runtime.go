package auth

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
)

var processAuth struct {
	mu      sync.RWMutex
	service *AuthService
	oidc    *OAuth2Client
	secure  bool
}

// Init builds authentication capabilities once, after DB, Redis and token
// infrastructure. Request routing does not construct authentication services.
func Init() error {
	redisClient, err := conn.ProcessRedis()
	if err != nil {
		return err
	}
	cfg := setting.WebConfig()
	phoneSender, err := notification.NewPhoneCodeSender(cfg)
	if err != nil {
		return err
	}
	mailer := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	service := NewAuthService(authcrypto.ProcessMaker(), cfg, phoneSender, mailer)
	var oidc *OAuth2Client
	var secure bool
	if cfg.OIDC.Enabled() {
		oidc, err = NewOAuth2Client(cfg.OIDC, redisClient, &http.Client{Timeout: 10 * time.Second})
		if err != nil {
			return err
		}
		secure = strings.HasPrefix(cfg.OIDC.RedirectURI, "https://")
	}
	processAuth.mu.Lock()
	defer processAuth.mu.Unlock()
	if processAuth.service != nil {
		return errors.New("auth: service already initialized")
	}
	processAuth.service, processAuth.oidc, processAuth.secure = service, oidc, secure
	return nil
}

func ProcessService() *AuthService {
	processAuth.mu.RLock()
	defer processAuth.mu.RUnlock()
	if processAuth.service == nil {
		panic("auth: service not initialized")
	}
	return processAuth.service
}

func ProcessOAuth2() (*OAuth2Client, bool) {
	processAuth.mu.RLock()
	defer processAuth.mu.RUnlock()
	return processAuth.oidc, processAuth.secure
}
