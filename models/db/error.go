package db

import "errors"

var (
	ErrEngineAlreadyInitialized = errors.New("db: engine already initialized")
	ErrEngineNotInitialized     = errors.New("db: engine not initialized")
	ErrEngineClosed             = errors.New("db: engine closed")
	ErrTransactionContext       = errors.New("db: invalid transaction context")
	ErrTransactionClosed        = errors.New("db: transaction closed")
	errTransactionPanicked      = errors.New("db: nested transaction callback panicked")
)
