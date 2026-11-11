package db

import "errors"

// Database infrastructure and transaction contract errors.
var errTransactionPanicked = errors.New("db: nested transaction callback panicked")

var (
	ErrStoreMissing         = errors.New("db: store missing from context")
	ErrTxStoreUnavailable   = errors.New("db: transaction-bound store unavailable")
	ErrTransactionRequired  = errors.New("db: transaction required")
	ErrTransactionClosed    = errors.New("db: transaction context is no longer active")
	ErrNestedTransaction    = errors.New("db: nested transaction context has no owner")
	ErrRawTransactionInSQLC = errors.New("db: raw transaction cannot nest inside SQLC transaction")
)

var (
	ErrEngineAlreadyInitialized = errors.New("db: engine already initialized")
	ErrEngineNotInitialized     = errors.New("db: engine not initialized")
	ErrEngineClosed             = errors.New("db: engine closed")
)
