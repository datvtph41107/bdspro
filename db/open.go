package database

import (
    "context"
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog/log"
)

// Open owns the PostgreSQL connection and schema migration startup.
func Open(ctx context.Context, source, migrationURL string)(*pgxpool.Pool,error){
    pool,err:=pgxpool.New(ctx,source)
    if err!=nil{return nil,fmt.Errorf("postgres connection: %w",err)}
    if err=pool.Ping(ctx);err!=nil{pool.Close();return nil,fmt.Errorf("postgres ping: %w",err)}
    log.Info().Msg("postgres.connected")
    m,err:=migrate.New(migrationURL,source)
    if err!=nil{pool.Close();return nil,fmt.Errorf("postgres migration init: %w",err)}
    err=m.Up()
    _,closeErr:=m.Close()
    if err!=nil && !errors.Is(err,migrate.ErrNoChange){pool.Close();return nil,fmt.Errorf("postgres migration: %w",err)}
    if closeErr!=nil{pool.Close();return nil,fmt.Errorf("close migration: %w",closeErr)}
    result:="applied";if errors.Is(err,migrate.ErrNoChange){result="no_change"}
    log.Info().Str("result",result).Msg("migration.checked")
    return pool,nil
}
