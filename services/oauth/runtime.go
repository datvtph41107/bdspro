package oauth

import (
	"errors"
	"sync"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
)

var processOAuth struct {
	mu      sync.RWMutex
	service *Service
	proofs  *ConsentProofs
}

// Init owns OAuth authorization server setup and proof signing material.
func Init() error {
	store, err := db.ProcessStore()
	if err != nil {
		return err
	}
	oauthStore, ok := store.(db.OAuthServerStore)
	if !ok {
		return errors.New("oauth: database does not support OAuth grants")
	}
	cfg := setting.WebConfig()
	proofs, err := NewConsentProofs(cfg.Authentication().JWTSecret)
	if err != nil {
		return err
	}
	service := NewService(oauthStore)
	processOAuth.mu.Lock()
	defer processOAuth.mu.Unlock()
	if processOAuth.service != nil {
		return errors.New("oauth: service already initialized")
	}
	processOAuth.service, processOAuth.proofs = service, proofs
	return nil
}

func ProcessService() (*Service, *ConsentProofs) {
	processOAuth.mu.RLock()
	defer processOAuth.mu.RUnlock()
	if processOAuth.service == nil {
		panic("oauth: service not initialized")
	}
	return processOAuth.service, processOAuth.proofs
}
