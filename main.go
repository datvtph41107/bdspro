package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	dev := os.Getenv("APP_ENV") == "development"
	host := os.Getenv("BIND_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	if dev && host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return fmt.Errorf("development verification tokens require a loopback listener")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	ping, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(ping)
	cancel()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	var mailer auth.Mailer
	if os.Getenv("SMTP_ADDR") != "" {
		mailer = auth.SMTPMailer{
			Addr: os.Getenv("SMTP_ADDR"), User: os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"), From: os.Getenv("SMTP_FROM"),
			VerificationURL: os.Getenv("VERIFICATION_URL"),
		}
	}
	if !dev && mailer == nil {
		return fmt.Errorf("SMTP_ADDR required outside development")
	}
	if !dev && os.Getenv("ALLOWED_ORIGIN") == "" {
		return fmt.Errorf("ALLOWED_ORIGIN required outside development")
	}
	authentication := &auth.Handler{
		S: auth.New(pool, mailer), Development: dev, SecureCookies: !dev,
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}
	organizations := &organization.Handler{Service: organization.New(pool), Auth: authentication}
	mux := http.NewServeMux()
	authentication.RegisterRoutes(mux)
	organizations.RegisterRoutes(mux)
	src := source.NewHandler(source.NewService(pool), authentication.Account)
	mux.Handle("/organizations/", src)
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := pool.Ping(c); err != nil {
			http.Error(w, "not ready", 503)
			return
		}
		_, _ = w.Write([]byte("ready"))
	})
	app := api.NewServer(dbsqlc.NewStore(pool), authentication, mux)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		if method != "GET" && method != "HEAD" && method != "OPTIONS" {
			site := strings.ToLower(r.Header.Get("Sec-Fetch-Site"))
			if site == "cross-site" {
				http.Error(w, "cross-site request denied", 403)
				return
			}
			origin := r.Header.Get("Origin")
			if origin != "" && origin != authentication.AllowedOrigin {
				http.Error(w, "origin not allowed", 403)
				return
			}
		}
		app.Handler().ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	address := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.Printf("BDSPro listening on %s", ln.Addr())
	signals, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ln) }()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-signals.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return err
		}
	}
	return nil
}
