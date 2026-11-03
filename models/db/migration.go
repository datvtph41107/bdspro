package db

import (
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp applies canonical versioned SQL migrations. steps=0 means all pending.
func MigrateUp(sourceURL, databaseURL string, steps int) (bool,error) {
    if steps < 0 {return false,fmt.Errorf("migration steps must not be negative")}
    m,err:=migrate.New(sourceURL,databaseURL)
    if err!=nil {return false,fmt.Errorf("open migrations: %w",err)}
    defer m.Close()
    if steps==0 {err=m.Up()} else {err=m.Steps(steps)}
    if errors.Is(err,migrate.ErrNoChange) {return false,nil}
    if err!=nil {return false,fmt.Errorf("apply migrations: %w",err)}
    return true,nil
}

// MigrateDown must be restricted to approved disposable development environments.
func MigrateDown(sourceURL,databaseURL string,steps int) (bool,error) {
    if steps < 0 {return false,fmt.Errorf("migration steps must not be negative")}
    m,err:=migrate.New(sourceURL,databaseURL)
    if err!=nil {return false,fmt.Errorf("open migrations: %w",err)}
    defer m.Close()
    if steps==0 {err=m.Down()} else {err=m.Steps(-steps)}
    if errors.Is(err,migrate.ErrNoChange) {return false,nil}
    if err!=nil {return false,fmt.Errorf("rollback migrations: %w",err)}
    return true,nil
}
