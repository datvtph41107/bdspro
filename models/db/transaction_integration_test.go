package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The value returned by WithTxResult must describe committed PostgreSQL rows,
// not merely a successful callback. Runs only against disposable migrated DB.
func TestWithTxResultPostgresAtomicity(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory for database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)

	var rolledBackID int64
	expectedErr := errors.New("abort after inserted Party")
	result, err := WithTxResult(ctx, store, func(q *Queries) (int64, error) {
		id, err := q.CreateParty(ctx)
		if err != nil {
			return 0, err
		}
		rolledBackID = id
		return id, expectedErr
	})
	if result != 0 || !errors.Is(err, expectedErr) {
		t.Fatalf("failed transaction published value=%d err=%v", result, err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties WHERE id=$1", rolledBackID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed transaction committed an unpaired Party")
	}

	committedID, err := WithTxResult(ctx, store, func(q *Queries) (int64, error) {
		return CreatePersonWithParty(ctx, q)
	})
	if err != nil || committedID <= 0 {
		t.Fatalf("commit result=%d err=%v", committedID, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM persons WHERE party_id=$1", committedID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("committed Person missing: party=%d count=%d", committedID, count)
	}
}

// TestWithTxNestedPostgresAtomicity verifies the same SQLC transaction is
// reused across nested services and that an ignored inner error still forces
// the outer PostgreSQL transaction to roll back. It uses the disposable DB.
func TestWithTxNestedPostgresAtomicity(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory for database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	base, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(base); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	ctx := WithStore(base, store)

	t.Run("one transaction commits nested changes", func(t *testing.T) {
		var created int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			outer, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			created, err = WithTx2(txCtx, func(inner context.Context) (int64, error) {
				nested, err := TxQueries(inner)
				if err != nil {
					return 0, err
				}
				if nested != outer {
					t.Fatal("nested SQLC executor differed from enclosing transaction")
				}
				return CreatePersonWithParty(inner, nested)
			})
			if err != nil {
				return err
			}
			// WithTx2 succeeded inside an uncommitted parent. Its returned
			// value is provisional until the parent WithTx commits.
			var visible int
			if err := pool.QueryRow(txCtx, "SELECT count(*) FROM persons WHERE party_id=$1", created).Scan(&visible); err != nil {
				return err
			}
			if visible != 0 {
				t.Fatal("nested writes were committed before outer transaction")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		var visible int
		if err := pool.QueryRow(base, "SELECT count(*) FROM persons WHERE party_id=$1", created).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != 1 {
			t.Fatalf("outer commit did not publish nested person: %d", visible)
		}
	})

	t.Run("ignored nested error poisons outer commit", func(t *testing.T) {
		sentinel := errors.New("nested operation rejected")
		var uncommittedID int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			uncommittedID, err = q.CreateParty(txCtx)
			if err != nil {
				return err
			}
			innerErr := WithTx(txCtx, func(inner context.Context) error {
				nested, err := TxQueries(inner)
				if err != nil {
					return err
				}
				if _, err := nested.CreateParty(inner); err != nil {
					return err
				}
				return sentinel
			})
			if !errors.Is(innerErr, sentinel) {
				return innerErr
			}
			// Simulate a buggy service that ignores the inner error.
			return nil
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("outer transaction committed despite nested failure: %v", err)
		}
		var count int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", uncommittedID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("ignored nested failure committed PostgreSQL data")
		}
	})
	t.Run("mixed entrypoints reuse one PostgreSQL transaction", func(t *testing.T) {
		sentinel := errors.New("force outer rollback")
		var partyID int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			outer, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			partyID, err = outer.CreateParty(txCtx)
			if err != nil {
				return err
			}
			_, err = WithTxResult(txCtx, store, func(q *Queries) (int64, error) {
				if q != outer {
					t.Fatal("WithTxResult used another SQLC executor")
				}
				return CreatePersonWithParty(txCtx, q)
			})
			if err != nil {
				return err
			}
			if err := store.ExecTx(txCtx, func(q *Queries) error {
				if q != outer {
					t.Fatal("SQLStore.ExecTx used another SQLC executor")
				}
				_, err := q.CreateParty(txCtx)
				return err
			}); err != nil {
				return err
			}
			var visible int
			if err := pool.QueryRow(txCtx, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&visible); err != nil {
				return err
			}
			if visible != 0 {
				t.Fatal("nested entrypoint committed before its parent")
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("outer rollback=%v", err)
		}
		var visible int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != 0 {
			t.Fatal("mixed nested entrypoints leaked a committed Party")
		}
	})

	t.Run("nested raw transaction refusal poisons outer commit", func(t *testing.T) {
		var partyID int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			partyID, err = q.CreateParty(txCtx)
			if err != nil {
				return err
			}
			inner := store.ExecTxRaw(txCtx, func(pgx.Tx) error {
				t.Fatal("raw transaction escaped the outer transaction")
				return nil
			})
			if !errors.Is(inner, ErrRawTransactionInSQLC) {
				t.Fatalf("unexpected raw transaction error: %v", inner)
			}
			return nil // parent must still roll back
		})
		if !errors.Is(err, ErrRawTransactionInSQLC) {
			t.Fatalf("outer did not preserve rollback-only failure: %v", err)
		}
		var visible int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != 0 {
			t.Fatal("SQLC outer transaction committed after raw nesting violation")
		}
	})

	t.Run("raw transaction and SQLC share the PostgreSQL owner", func(t *testing.T) {
		var partyID int64
		sentinel := errors.New("reject raw transaction")
		err := store.ExecTxRaw(ctx, func(tx pgx.Tx) error {
			q := New(tx)
			var err error
			partyID, err = q.CreateParty(ctx)
			if err != nil {
				return err
			}
			var visible int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&visible); err != nil {
				return err
			}
			if visible != 1 {
				t.Fatal("transaction-scoped raw reader did not see SQLC write")
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("raw transaction error=%v", err)
		}
		var visible int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != 0 {
			t.Fatal("raw transaction rolled back but SQLC write committed")
		}
	})

	t.Run("caught nested panic poisons PostgreSQL rollback", func(t *testing.T) {
		var partyID int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			partyID, err = q.CreateParty(txCtx)
			if err != nil {
				return err
			}
			func() {
				defer func() {
					if got := recover(); got != "caught inner panic" {
						t.Fatalf("recovered=%v, want caught inner panic", got)
					}
				}()
				_ = WithTx(txCtx, func(innerCtx context.Context) error {
					nested, err := TxQueries(innerCtx)
					if err != nil {
						return err
					}
					if _, err = nested.CreateParty(innerCtx); err != nil {
						return err
					}
					panic("caught inner panic")
				})
			}()
			return nil // malicious or buggy caller catches panic and tries commit
		})
		if !errors.Is(err, errTransactionPanicked) {
			t.Fatalf("outer did not roll back after recovered nested panic: %v", err)
		}
		var count int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", partyID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("recovered nested panic committed a PostgreSQL Party")
		}
	})

	t.Run("transaction context is invalid after PostgreSQL owner exits", func(t *testing.T) {
		for _, shouldRollback := range []bool{false, true} {
			var escaped context.Context
			sentinel := errors.New("rollback completed transaction")
			err := WithTx(ctx, func(txCtx context.Context) error {
				escaped = txCtx
				q, err := TxQueries(txCtx)
				if err != nil {
					return err
				}
				if _, err := q.CreateParty(txCtx); err != nil {
					return err
				}
				if shouldRollback {
					return sentinel
				}
				return nil
			})
			if shouldRollback {
				if !errors.Is(err, sentinel) {
					t.Fatalf("rollback err=%v", err)
				}
			} else if err != nil {
				t.Fatalf("commit err=%v", err)
			}
			if q, err := TxQueries(escaped); q != nil || !errors.Is(err, ErrTransactionClosed) {
				t.Fatalf("escaped executor after owner exit: q=%p err=%v", q, err)
			}
			if err := WithTx(escaped, func(context.Context) error {
				t.Fatal("closed transaction context ran callback")
				return nil
			}); !errors.Is(err, ErrTransactionClosed) {
				t.Fatalf("closed transaction callback error=%v", err)
			}
		}
	})

	t.Run("ingress Store direct SQLC calls follow transaction owner", func(t *testing.T) {
		abort := errors.New("abort after direct ingress Store calls")
		var partyID int64
		var escaped context.Context
		err := WithTx(ctx, func(txCtx context.Context) error {
			escaped = txCtx
			// Deliberately keep using the ingress Store, rather than retrieving
			// StoreFromContext. All calls still receive the correct txCtx.
			var err error
			partyID, err = store.CreateParty(txCtx) // QueryRow
			if err != nil {
				return err
			}
			if err := store.CreatePerson(txCtx, partyID); err != nil { // Exec
				return err
			}
			if _, err := store.ListPendingJobOutbox(txCtx, 1); err != nil { // Query
				return err
			}
			var outside int
			if err := pool.QueryRow(txCtx, "SELECT count(*) FROM persons WHERE party_id=$1", partyID).Scan(&outside); err != nil {
				return err
			}
			if outside != 0 {
				t.Fatal("ingress Store committed a SQLC write outside parent transaction")
			}
			return abort
		})
		if !errors.Is(err, abort) || partyID <= 0 {
			t.Fatalf("rollback=%v partyID=%d", err, partyID)
		}
		var persisted int
		if err := pool.QueryRow(base, "SELECT count(*) FROM persons WHERE party_id=$1", partyID).Scan(&persisted); err != nil {
			t.Fatal(err)
		}
		if persisted != 0 {
			t.Fatal("ingress Store bypassed the transaction owner")
		}
		if _, err := store.CreateParty(escaped); !errors.Is(err, ErrTransactionClosed) {
			t.Fatalf("ingress Store accepted escaped txCtx: %v", err)
		}
	})

	t.Run("cross-pool ingress Store cannot hijack transaction", func(t *testing.T) {
		otherPool, err := pgxpool.New(base, source)
		if err != nil {
			t.Fatal(err)
		}
		defer otherPool.Close()
		foreignStore := NewStore(otherPool)
		abort := ErrTxStoreUnavailable
		var parentID int64
		err = WithTx(ctx, func(txCtx context.Context) error {
			var err error
			parentID, err = store.CreateParty(txCtx)
			if err != nil {
				return err
			}
			_, refused := foreignStore.CreateParty(txCtx)
			if !errors.Is(refused, abort) {
				t.Fatalf("foreign Store used another pool's transaction: %v", refused)
			}
			return nil // Even swallowed cross-pool access must abort parent.
		})
		if !errors.Is(err, abort) {
			t.Fatalf("cross-pool failure did not poison owner: %v", err)
		}
		var count int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", parentID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("wrong-pool access left parent write committed")
		}

		err = WithTx(ctx, func(txCtx context.Context) error {
			// Replacing the context's Store must not create a valid domain
			// accessor for another pool while keeping the old tx executor.
			crossCtx := WithStore(txCtx, foreignStore)
			if other, err := StoreFromContext(crossCtx); other != nil || !errors.Is(err, abort) {
				t.Fatalf("StoreFromContext accepted wrong pool: store=%v err=%v", other, err)
			}
			return nil
		})
		if !errors.Is(err, abort) {
			t.Fatalf("wrong-pool StoreFromContext did not abort parent: %v", err)
		}

		// A nested WithTx also must reject an overridden foreign Store,
		// even when its callback does not explicitly call a query.
		err = WithTx(ctx, func(txCtx context.Context) error {
			crossCtx := WithStore(txCtx, foreignStore)
			inner := WithTx(crossCtx, func(context.Context) error {
				t.Fatal("nested transaction accepted a foreign Store owner")
				return nil
			})
			if !errors.Is(inner, abort) {
				t.Fatalf("foreign nested Store accepted: %v", inner)
			}
			return nil // Ignoring this violation still must poison parent.
		})
		if !errors.Is(err, abort) {
			t.Fatalf("foreign nested WithTx did not poison parent: %v", err)
		}
	})

	t.Run("second Store on same pool shares transaction owner", func(t *testing.T) {
		otherStore := NewStore(pool)
		abort := errors.New("rollback same-pool Store")
		var id int64
		err := WithTx(ctx, func(txCtx context.Context) error {
			// A different Store instance for the SAME pool must reuse
			// the existing executor and must not start a second owner.
			if err := WithTx(WithStore(txCtx, otherStore), func(inner context.Context) error {
				parentQ, err := TxQueries(txCtx)
				if err != nil {
					return err
				}
				innerQ, err := TxQueries(inner)
				if err != nil {
					return err
				}
				if innerQ != parentQ {
					t.Fatal("same-pool nested transaction changed SQLC executor")
				}
				return nil
			}); err != nil {
				return err
			}
			var err error
			id, err = otherStore.CreateParty(txCtx)
			if err != nil {
				return err
			}
			return abort
		})
		if !errors.Is(err, abort) {
			t.Fatalf("same-pool Store query lost owner: %v", err)
		}
		var count int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("second Store on same pool committed independently")
		}
	})

	t.Run("transaction-scoped Store follows owner rollback", func(t *testing.T) {
		abort := errors.New("rollback owning transaction")
		var created int64
		var scoped Store
		var escaped context.Context
		err := WithTx(ctx, func(txCtx context.Context) error {
			escaped = txCtx
			owner, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			scoped, err = StoreFromContext(txCtx)
			if err != nil {
				return err
			}
			view, ok := scoped.(*SQLStore)
			if !ok || view.Queries != owner || view.pool != nil {
				t.Fatal("StoreFromContext returned pool executor instead of active transaction")
			}
			// Emulate existing model code calling a Store query rather than
			// TxQueries directly. It must still write to the same executor.
			created, err = scoped.CreateParty(txCtx)
			if err != nil {
				return err
			}
			var ownerVisible int
			if err := pool.QueryRow(txCtx, "SELECT count(*) FROM parties WHERE id=$1", created).Scan(&ownerVisible); err != nil {
				return err
			}
			if ownerVisible != 0 {
				t.Fatal("scoped Store wrote outside the owning PostgreSQL transaction")
			}
			return abort
		})
		if !errors.Is(err, abort) || created <= 0 {
			t.Fatalf("parent rollback err=%v id=%d", err, created)
		}
		var persisted int
		if err := pool.QueryRow(base, "SELECT count(*) FROM parties WHERE id=$1", created).Scan(&persisted); err != nil {
			t.Fatal(err)
		}
		if persisted != 0 {
			t.Fatal("StoreFromContext committed a write outside parent transaction")
		}
		if got, err := StoreFromContext(escaped); got != nil || !errors.Is(err, ErrTransactionClosed) {
			t.Fatalf("StoreFromContext exposed expired owner: store=%v err=%v", got, err)
		}
		if err := scoped.ExecTx(base, func(*Queries) error {
			t.Fatal("expired scoped Store started another transaction")
			return nil
		}); !errors.Is(err, ErrTransactionClosed) {
			t.Fatalf("escaped Store opened a separate SQLC transaction: %v", err)
		}
	})
}

func TestExecTxRawCancellationRollsBackAndReleasesConnection(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("BDS_TEST_MODE=disposable is mandatory for database writes")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write outside bdspro_test")
	}
	// A single-connection pool detects an owner that fails to release its tx.
	cfg.MaxConns = 1
	base, stop := context.WithTimeout(context.Background(), 20*time.Second)
	defer stop()
	pool, err := pgxpool.NewWithConfig(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(base); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	requestCtx, cancel := context.WithCancel(base)
	defer cancel()
	var uncommittedID int64
	err = store.ExecTxRaw(requestCtx, func(tx pgx.Tx) error {
		q := New(tx)
		var insertErr error
		uncommittedID, insertErr = q.CreateParty(requestCtx)
		if insertErr != nil {
			return insertErr
		}
		cancel()
		// Simulate a canceled request after an in-transaction write.
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled raw owner returned %v", err)
	}
	if uncommittedID <= 0 {
		t.Fatal("test did not insert a party before cancellation")
	}
	// If the canceled context were reused for ROLLBACK, cleanup can fail;
	// a fresh, bounded context must be able to acquire the sole connection.
	checkCtx, checkCancel := context.WithTimeout(base, 5*time.Second)
	defer checkCancel()
	var count int
	if err := pool.QueryRow(checkCtx, "SELECT count(*) FROM parties WHERE id=$1", uncommittedID).Scan(&count); err != nil {
		t.Fatalf("raw transaction did not release connection after cancel: %v", err)
	}
	if count != 0 {
		t.Fatal("canceled raw transaction committed its PostgreSQL write")
	}
}
