package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/listing"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer db.Close()

	pingCtx, cancelPing := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	err = db.Ping(pingCtx)
	cancelPing()

	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.Println("database ready")

	handler := newHTTPHandler(db)

	server := &http.Server{
		Handler: handler,
	}

	addr := ":" + port

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	log.Printf("bdspro listening on %s", listener.Addr())

	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server failed: %w", err)
		}

	case <-signalCtx.Done():
		log.Println("shutdown signal received")

		shutdownCtx, cancelShutdown := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelShutdown()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		log.Println("bdspro stopped")
	}

	return nil
}

func newHTTPHandler(db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(
				w,
				"not ready",
				http.StatusServiceUnavailable,
			)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.HandleFunc("GET /listings/{id}/publication-readiness", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(
			r.PathValue("id"),
			10,
			64,
		)
		if err != nil || id <= 0 {
			http.Error(
				w,
				"invalid listing id",
				http.StatusBadRequest,
			)
			return
		}

		readiness, err := listing.CheckPublicationReadiness(
			r.Context(),
			db,
			id,
		)

		switch {
		case errors.Is(err, listing.ErrNotFound):
			http.Error(
				w,
				"listing not found",
				http.StatusNotFound,
			)
			return

		case err != nil:
			log.Printf(
				"check listing publication readiness: %v",
				err,
			)

			http.Error(
				w,
				"check publication readiness failed",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		if err := json.NewEncoder(w).Encode(readiness); err != nil {
			log.Printf(
				"encode publication readiness: %v",
				err,
			)
		}
	},
	)

	mux.HandleFunc("POST /listings", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(
				w,
				"invalid request body",
				http.StatusBadRequest,
			)
			return
		}

		result, err := listing.Create(
			r.Context(),
			db,
			input.Title,
		)

		switch {
		case errors.Is(err, listing.ErrTitleRequired):
			http.Error(
				w,
				"title is required",
				http.StatusBadRequest,
			)
			return

		case err != nil:
			log.Printf("listing create failed: %v", err)

			http.Error(
				w,
				"listing create failed",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set(
			"Content-Type",
			"application/json",
		)
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf(
				"encode create listing response: %v",
				err,
			)
		}
	})

	mux.HandleFunc("PATCH /listings/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid listing id", http.StatusBadRequest)
			return
		}

		var input struct {
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		result, err := listing.UpdateDescription(
			r.Context(),
			db,
			id,
			input.Description,
		)

		switch {
		case errors.Is(err, listing.ErrDescriptionRequired):
			http.Error(
				w,
				"description is required",
				http.StatusBadRequest,
			)
			return

		case errors.Is(err, listing.ErrNotFound):
			http.Error(
				w,
				"listing not found",
				http.StatusNotFound,
			)
			return

		case errors.Is(err, listing.ErrNotEditable):
			http.Error(
				w,
				"listing is not editable",
				http.StatusConflict,
			)
			return

		case err != nil:
			log.Printf("update listing description: %v", err)

			http.Error(
				w,
				"update listing failed",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("encode listing response: %v", err)
		}
	})

	mux.HandleFunc("POST /listings/{id}/publish", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			http.Error(
				w,
				"invalid listing id",
				http.StatusBadRequest,
			)
			return
		}

		result, err := listing.Publish(
			r.Context(),
			db,
			id,
		)

		switch {
		case errors.Is(err, listing.ErrNotFound):
			http.Error(
				w,
				"listing not found",
				http.StatusNotFound,
			)
			return

		case errors.Is(err, listing.ErrAlreadyPublished):
			http.Error(
				w,
				"listing already published",
				http.StatusConflict,
			)
			return

		case errors.Is(err, listing.ErrNotPublishable):
			http.Error(
				w,
				"listing is not publishable",
				http.StatusConflict,
			)
			return

		case err != nil:
			log.Printf("publish listing: %v", err)

			http.Error(
				w,
				"publish listing failed",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf(
				"encode publish listing response: %v",
				err,
			)
		}
	},
	)

	return mux
}
