package db

import (
	"context"
	"testing"
)

func TestConnectRejectsMissingPostgresSource(t *testing.T) {
	if _, err := Connect(context.Background(), ""); err == nil {
		t.Fatal("missing database source accepted")
	}
}

func TestInitStoreRejectsMissingPostgresSource(t *testing.T) {
	if _, err := InitStore(context.Background(), ""); err == nil {
		t.Fatal("missing database source accepted")
	}
}
