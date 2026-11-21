package auth

import (
	"errors"
	"sync"
)

var processMaker struct {
	mu    sync.RWMutex
	maker Maker
}

// Init initializes the process token maker from already-loaded settings.
func Init(secretKey string) error {
	maker, err := NewJWTMaker(secretKey)
	if err != nil {
		return err
	}
	processMaker.mu.Lock()
	defer processMaker.mu.Unlock()
	if processMaker.maker != nil {
		return errors.New("auth: token maker already initialized")
	}
	processMaker.maker = maker
	return nil
}

func ProcessMaker() Maker {
	processMaker.mu.RLock()
	defer processMaker.mu.RUnlock()
	if processMaker.maker == nil {
		panic("auth: token maker not initialized")
	}
	return processMaker.maker
}
