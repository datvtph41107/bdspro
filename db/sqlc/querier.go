package db

import (
    "context"
    "github.com/google/uuid"
)

type Querier interface {
    CreatePersonParty(context.Context) (int64, error)
    CreatePerson(context.Context, int64) error
    CreateAccount(context.Context, CreateAccountParams) (Account, error)
    GetAccountByEmail(context.Context, string) (Account, error)
    CreateSession(context.Context, CreateSessionParams) (Session, error)
    GetSession(context.Context, uuid.UUID) (Session, error)
}

var _ Querier = (*Queries)(nil)
