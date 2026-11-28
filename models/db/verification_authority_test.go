package db

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestVerificationAuthorityAfterLock(t *testing.T) {
	source := os.Getenv("BDSPRO_VERIFICATION_AUTHORITY_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_VERIFICATION_AUTHORITY_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	locker, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer locker.Close(context.Background())
	waiter, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer waiter.Close(context.Background())
	digest := func() []byte {
		sum := sha256.Sum256([]byte(uuid.NewString()))
		return sum[:]
	}
	setup, err := locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Rollback(context.Background())
	q := sqlc.New(setup)
	person, err := q.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.CreatePerson(ctx, person); err != nil {
		t.Fatal(err)
	}
	account, err := q.CreateAccountWithoutEmail(ctx, person)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	if _, err := q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID: sessionID, AccountID: account.ID, RefreshToken: "fixture",
		UserAgent: "fixture", ClientIp: "127.0.0.1", ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, statement := range []struct {
			sql string
			id  any
		}{
			{"DELETE FROM google_flows WHERE account_id=$1", account.ID},
			{"DELETE FROM sessions WHERE account_id=$1", account.ID},
			{"DELETE FROM accounts WHERE id=$1", account.ID},
			{"DELETE FROM persons WHERE party_id=$1", person},
			{"DELETE FROM parties WHERE id=$1", person},
		} {
			if _, err := locker.Exec(cleanup, statement.sql, statement.id); err != nil {
				t.Error("fixture cleanup failed", err)
			}
		}
	}()

	// A dedicated connection holds the fixture row until the second connection
	// is demonstrably blocked. Only then wait for the DB clock to cross expiry.
	releaseAfterWait := func(block pgx.Tx, deadline time.Time) <-chan error {
		result := make(chan error, 1)
		lockerPID, waiterPID := int32(locker.PgConn().PID()), int32(waiter.PgConn().PID())
		go func() {
			result <- func() error {
				defer block.Rollback(context.Background())
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for {
					var waiting bool
					if err := block.QueryRow(ctx, "SELECT $2::integer=ANY(pg_blocking_pids($1::integer))", waiterPID, lockerPID).Scan(&waiting); err != nil {
						return err
					}
					if waiting {
						break
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-ticker.C:
					}
				}
				if !deadline.IsZero() {
					if _, err := block.Exec(ctx, "SELECT pg_sleep(GREATEST(0,EXTRACT(EPOCH FROM ($1::timestamptz-clock_timestamp()))::double precision+0.05))", deadline); err != nil {
						return err
					}
				}
				return block.Commit(ctx)
			}()
		}()
		return result
	}
	assertRejected := func(t *testing.T, block pgx.Tx, deadline time.Time, read func(*sqlc.Queries) error) {
		t.Helper()
		released := releaseAfterWait(block, deadline)
		err := read(sqlc.New(waiter))
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("row-lock wait not verified", releaseErr)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("expired or consumed authority admitted", err)
		}
	}

	for _, kind := range []string{"email", "phone"} {
		t.Run(kind+" proof", func(t *testing.T) {
			id, browser, proof := uuid.New(), digest(), digest()
			destination := "fixture@example.invalid"
			if kind == "phone" {
				destination = "+84912345678"
			}
			var deadline time.Time
			if err := locker.QueryRow(ctx, "INSERT INTO verifications(kind,id,destination,browser_digest,proof_digest,expires_at,resend_after) VALUES ($1,$2,$3,$4,$5,clock_timestamp()+interval '3 seconds',clock_timestamp()) RETURNING expires_at", kind, id, destination, browser, proof).Scan(&deadline); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := locker.Exec(ctx, "DELETE FROM verifications WHERE kind=$1 AND id=$2", kind, id); err != nil {
					t.Error(err)
				}
			}()
			read := func(q *sqlc.Queries) error {
				if kind == "email" {
					_, err := q.LockEmailRegistrationProof(ctx, sqlc.LockEmailRegistrationProofParams{ProofDigest: proof, BrowserDigest: browser})
					return err
				}
				_, err := q.LockPhoneVerificationProof(ctx, sqlc.LockPhoneVerificationProofParams{ProofDigest: proof, BrowserDigest: browser})
				return err
			}
			block, err := locker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer block.Rollback(context.Background())
			if _, err := block.Exec(ctx, "SELECT id FROM verifications WHERE kind=$1 AND id=$2 FOR UPDATE", kind, id); err != nil {
				t.Fatal(err)
			}
			assertRejected(t, block, deadline, read)
			var count int
			if err := locker.QueryRow(ctx, "SELECT count(*) FROM verifications WHERE kind=$1 AND id=$2", kind, id).Scan(&count); err != nil || count != 1 {
				t.Fatal("expired proof removed", err)
			}
			if _, err := locker.Exec(ctx, "UPDATE verifications SET expires_at=clock_timestamp()+interval '10 minutes' WHERE kind=$1 AND id=$2", kind, id); err != nil {
				t.Fatal(err)
			}
			block, err = locker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer block.Rollback(context.Background())
			if err := read(sqlc.New(block)); err != nil {
				t.Fatal("valid proof denied", err)
			}
			if kind == "email" {
				err = sqlc.New(block).DeleteEmailRegistration(ctx, id)
			} else {
				err = sqlc.New(block).DeletePhoneVerification(ctx, id)
			}
			if err != nil {
				t.Fatal(err)
			}
			assertRejected(t, block, time.Time{}, read)
			t.Log("PASS: expiry after lock and concurrent proof replay denied")
		})
	}

	for _, mode := range []string{"ready", "registration", "authenticated"} {
		t.Run("google "+mode, func(t *testing.T) {
			browser, proof := digest(), digest()
			phase, purpose, action := mode, "login", ""
			var proofArg []byte
			var accountArg any
			var sessionArg any
			if mode == "registration" {
				proofArg = proof
			}
			if mode == "authenticated" {
				phase, purpose, action = "ready", "link", "google.link"
				accountArg, sessionArg = account.ID, sessionID
			}
			var deadline time.Time
			if err := locker.QueryRow(ctx, "INSERT INTO google_flows(browser_digest,phase,subject,remember,proof_digest,expires_at,account_id,session_id,purpose,action) VALUES ($1,$2,'fixture-subject',false,$3,clock_timestamp()+interval '3 seconds',$4,$5,$6,$7) RETURNING expires_at", browser, phase, proofArg, accountArg, sessionArg, purpose, action).Scan(&deadline); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := sqlc.New(locker).DeleteGoogleFlow(ctx, browser); err != nil {
					t.Error(err)
				}
			}()
			read := func(q *sqlc.Queries) error {
				switch mode {
				case "ready":
					_, err := q.LockGoogleResult(ctx, browser)
					return err
				case "registration":
					_, err := q.LockGoogleRegistration(ctx, sqlc.LockGoogleRegistrationParams{BrowserDigest: browser, ProofDigest: proof})
					return err
				default:
					_, err := q.LockGoogleAuthenticatedResult(ctx, sqlc.LockGoogleAuthenticatedResultParams{BrowserDigest: browser, AccountID: account.ID, SessionID: sessionID, Purpose: purpose})
					return err
				}
			}
			block, err := locker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer block.Rollback(context.Background())
			if _, err := block.Exec(ctx, "SELECT browser_digest FROM google_flows WHERE browser_digest=$1 FOR UPDATE", browser); err != nil {
				t.Fatal(err)
			}
			assertRejected(t, block, deadline, read)
			var count int
			if err := locker.QueryRow(ctx, "SELECT count(*) FROM google_flows WHERE browser_digest=$1", browser).Scan(&count); err != nil || count != 1 {
				t.Fatal("expired flow unexpectedly removed", err)
			}
			if _, err := locker.Exec(ctx, "UPDATE google_flows SET expires_at=clock_timestamp()+interval '10 minutes' WHERE browser_digest=$1", browser); err != nil {
				t.Fatal(err)
			}
			if err := read(sqlc.New(waiter)); err != nil {
				t.Fatal("valid flow denied", err)
			}
			t.Log("PASS: expired flow rejected after confirmed wait; valid flow admitted")
		})
	}
	t.Run("google pending claim", func(t *testing.T) {
		browser, state := digest(), digest()
		if err := sqlc.New(locker).CreateGoogleFlow(ctx, sqlc.CreateGoogleFlowParams{
			BrowserDigest: browser, StateDigest: state,
			Nonce:    "0123456789012345678901234567890123456789012",
			Verifier: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ",
		}); err != nil {
			t.Fatal(err)
		}
		defer sqlc.New(locker).DeleteGoogleFlow(ctx, browser)
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		params := sqlc.LockPendingGoogleFlowParams{BrowserDigest: browser, StateDigest: state}
		if _, err := sqlc.New(block).LockPendingGoogleFlow(ctx, params); err != nil {
			t.Fatal(err)
		}
		if n, err := sqlc.New(block).ClaimGoogleFlow(ctx, browser); err != nil || n != 1 {
			t.Fatal("first pending claim failed", err)
		}
		assertRejected(t, block, time.Time{}, func(q *sqlc.Queries) error {
			_, err := q.LockPendingGoogleFlow(ctx, params)
			return err
		})
		t.Log("PASS: concurrent pending callback cannot claim twice")
	})

	t.Run("google finish", func(t *testing.T) {
		browser := digest()
		var deadline time.Time
		if err := locker.QueryRow(ctx, "INSERT INTO google_flows(browser_digest,phase,remember,expires_at) VALUES ($1,'exchanging',false,clock_timestamp()+interval '3 seconds') RETURNING expires_at", browser).Scan(&deadline); err != nil {
			t.Fatal(err)
		}
		defer sqlc.New(locker).DeleteGoogleFlow(ctx, browser)
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		if _, err := block.Exec(ctx, "SELECT browser_digest FROM google_flows WHERE browser_digest=$1 FOR UPDATE", browser); err != nil {
			t.Fatal(err)
		}
		params := sqlc.FinishGoogleFlowParams{BrowserDigest: browser, Subject: "fixture-subject"}
		released := releaseAfterWait(block, deadline)
		n, err := sqlc.New(waiter).FinishGoogleFlow(ctx, params)
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("exchange lock wait not verified", releaseErr)
		}
		if err != nil || n != 0 {
			t.Fatal("expired exchange finished", n, err)
		}
		var phase, subject string
		if err := locker.QueryRow(ctx, "SELECT phase,subject FROM google_flows WHERE browser_digest=$1", browser).Scan(&phase, &subject); err != nil || phase != "exchanging" || subject != "" {
			t.Fatal("expired exchange mutated", err)
		}
		if _, err := locker.Exec(ctx, "UPDATE google_flows SET expires_at=clock_timestamp()+interval '2 minutes' WHERE browser_digest=$1", browser); err != nil {
			t.Fatal(err)
		}
		if n, err := sqlc.New(waiter).FinishGoogleFlow(ctx, params); err != nil || n != 1 {
			t.Fatal("valid exchange denied", n, err)
		}
		if n, err := sqlc.New(waiter).FinishGoogleFlow(ctx, params); err != nil || n != 0 {
			t.Fatal("exchange finished twice", n, err)
		}
		t.Log("PASS: exchange completion rejects post-lock expiry and duplicate finish")
	})

	for _, kind := range []string{"email", "phone"} {
		t.Run(kind+" OTP concurrency", func(t *testing.T) {
			id, browser, originalMAC := uuid.New(), digest(), digest()
			destination := "fixture@example.invalid"
			if kind == "phone" {
				destination = "+84912345678"
			}
			if _, err := locker.Exec(ctx, "INSERT INTO verifications(kind,id,destination,browser_digest,code_mac,expires_at,resend_after,failed_attempts) VALUES ($1,$2,$3,$4,$5,clock_timestamp()+interval '5 minutes',clock_timestamp(),3)", kind, id, destination, browser, originalMAC); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := locker.Exec(ctx, "DELETE FROM verifications WHERE kind=$1 AND id=$2", kind, id); err != nil {
					t.Error(err)
				}
			}()
			rotate := func(q *sqlc.Queries, mac []byte) (int64, error) {
				if kind == "email" {
					return q.RotateEmailRegistration(ctx, sqlc.RotateEmailRegistrationParams{ID: id, CodeMac: mac, TtlMinutes: 5, CooldownSeconds: 60, MaxFailedAttempts: 5, ResendHorizonMinutes: 30})
				}
				return q.RotatePhoneVerification(ctx, sqlc.RotatePhoneVerificationParams{ID: id, CodeMac: mac, TtlMinutes: 5, CooldownSeconds: 60, MaxFailedAttempts: 5, ResendHorizonMinutes: 30})
			}
			fail := func(q *sqlc.Queries) error {
				if kind == "email" {
					return q.FailEmailRegistration(ctx, sqlc.FailEmailRegistrationParams{ID: id, MaxFailedAttempts: 5})
				}
				return q.FailPhoneVerification(ctx, sqlc.FailPhoneVerificationParams{ID: id, MaxFailedAttempts: 5})
			}
			firstMAC := digest()
			block, err := locker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer block.Rollback(context.Background())
			if n, err := rotate(sqlc.New(block), firstMAC); err != nil || n != 1 {
				t.Fatal("first resend failed", n, err)
			}
			released := releaseAfterWait(block, time.Time{})
			n, err := rotate(sqlc.New(waiter), digest())
			if releaseErr := <-released; releaseErr != nil {
				t.Fatal("resend contention not verified", releaseErr)
			}
			if err != nil || n != 0 {
				t.Fatal("concurrent resend bypassed cooldown", n, err)
			}
			var attempts int
			var mac []byte
			if err := locker.QueryRow(ctx, "SELECT failed_attempts,code_mac FROM verifications WHERE kind=$1 AND id=$2", kind, id).Scan(&attempts, &mac); err != nil || attempts != 3 || !bytes.Equal(mac, firstMAC) {
				t.Fatal("resend overwrote winning MAC or failed attempts", err)
			}
			if _, err := locker.Exec(ctx, "UPDATE verifications SET failed_attempts=4,resend_after=clock_timestamp()-interval '1 second' WHERE kind=$1 AND id=$2", kind, id); err != nil {
				t.Fatal(err)
			}
			block, err = locker.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer block.Rollback(context.Background())
			if err := fail(sqlc.New(block)); err != nil {
				t.Fatal(err)
			}
			released = releaseAfterWait(block, time.Time{})
			err = fail(sqlc.New(waiter))
			if releaseErr := <-released; releaseErr != nil {
				t.Fatal("attempt contention not verified", releaseErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := locker.QueryRow(ctx, "SELECT failed_attempts FROM verifications WHERE kind=$1 AND id=$2", kind, id).Scan(&attempts); err != nil || attempts != 5 {
				t.Fatal("failed attempt cap violated", attempts, err)
			}
			if n, err := rotate(sqlc.New(waiter), digest()); err != nil || n != 0 {
				t.Fatal("resend bypassed attempt cap", n, err)
			}
			t.Log("PASS: one resend, failed-attempt state preserved, concurrent cap enforced")
		})
	}

	t.Run("contact link binding and contention", func(t *testing.T) {
		params := sqlc.StartCredentialLinkParams{
			ID: uuid.New(), AccountID: account.ID, SessionID: sessionID, Kind: "email",
			Destination: "fixture@example.invalid", CodeMac: digest(),
			TtlMinutes: 5, CooldownSeconds: 60, MaxFailedAttempts: 5,
		}
		if n, err := sqlc.New(locker).StartCredentialLink(ctx, params); err != nil || n != 1 {
			t.Fatal("challenge setup failed", n, err)
		}
		defer sqlc.New(locker).DeleteAccountCredentialLinks(ctx, account.ID)
		binding := sqlc.LockCredentialLinkParams{ID: params.ID, AccountID: account.ID, SessionID: sessionID, Kind: "email"}
		for _, wrong := range []sqlc.LockCredentialLinkParams{
			{ID: binding.ID, AccountID: account.ID + 1, SessionID: sessionID, Kind: "email"},
			{ID: binding.ID, AccountID: account.ID, SessionID: uuid.New(), Kind: "email"},
			{ID: binding.ID, AccountID: account.ID, SessionID: sessionID, Kind: "phone"},
		} {
			if _, err := sqlc.New(waiter).LockCredentialLink(ctx, wrong); !errors.Is(err, pgx.ErrNoRows) {
				t.Fatal("wrong challenge binding accepted", err)
			}
		}
		if _, err := locker.Exec(ctx, "UPDATE contact_link_challenges SET failed_attempts=3,resend_after=clock_timestamp()-interval '1 second' WHERE id=$1", params.ID); err != nil {
			t.Fatal(err)
		}
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		first := params
		first.ID, first.CodeMac = uuid.New(), digest()
		first.Destination = "changed@example.invalid"
		if n, err := sqlc.New(block).StartCredentialLink(ctx, first); err != nil || n != 1 {
			t.Fatal("first replacement failed", n, err)
		}
		second := params
		second.ID, second.CodeMac = uuid.New(), digest()
		released := releaseAfterWait(block, time.Time{})
		n, err := sqlc.New(waiter).StartCredentialLink(ctx, second)
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("contact contention not verified", releaseErr)
		}
		if err != nil || n != 0 {
			t.Fatal("concurrent contact replacement bypassed cooldown", n, err)
		}
		if _, err := sqlc.New(waiter).LockCredentialLink(ctx, binding); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("old challenge id still usable", err)
		}
		binding.ID = first.ID
		row, err := sqlc.New(waiter).LockCredentialLink(ctx, binding)
		if err != nil || row.FailedAttempts != 3 || row.Destination != first.Destination || !bytes.Equal(row.CodeMac, first.CodeMac) {
			t.Fatal("replacement lost attempt or destination binding", err)
		}
		if _, err := locker.Exec(ctx, "UPDATE contact_link_challenges SET failed_attempts=5,expires_at=clock_timestamp()-interval '1 second',resend_after=clock_timestamp()-interval '1 second' WHERE id=$1", first.ID); err != nil {
			t.Fatal(err)
		}
		if n, err := sqlc.New(waiter).StartCredentialLink(ctx, second); err != nil || n != 1 {
			t.Fatal("expired challenge could not reset", n, err)
		}
		binding.ID = second.ID
		row, err = sqlc.New(waiter).LockCredentialLink(ctx, binding)
		if err != nil || row.FailedAttempts != 0 || row.Destination != second.Destination || !bytes.Equal(row.CodeMac, second.CodeMac) {
			t.Fatal("expired challenge reset incorrect", err)
		}
		t.Log("PASS: contact binding, concurrent replacement, preserved attempts and expired reset")
	})

	t.Run("canonical Google identity contention", func(t *testing.T) {
		issuer, subject := "https://accounts.google.com", uuid.NewString()
		block, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer block.Rollback(context.Background())
		params := sqlc.LinkOAuth2CredentialParams{Issuer: issuer, Subject: subject, AccountID: account.ID}
		if owner, err := sqlc.New(block).LinkOAuth2Credential(ctx, params); err != nil || owner != account.ID {
			t.Fatal("first identity claim failed", err)
		}
		tx, err := waiter.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		q := sqlc.New(tx)
		losingPerson, err := q.CreateParty(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := q.CreatePerson(ctx, losingPerson); err != nil {
			t.Fatal(err)
		}
		loser, err := q.CreateAccountWithoutEmail(ctx, losingPerson)
		if err != nil {
			t.Fatal(err)
		}
		params.AccountID = loser.ID
		released := releaseAfterWait(block, time.Time{})
		_, err = q.LinkOAuth2Credential(ctx, params)
		if releaseErr := <-released; releaseErr != nil {
			t.Fatal("identity contention not verified", releaseErr)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("second account claimed canonical subject", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		owner, err := sqlc.New(locker).GetOAuth2AccountID(ctx, sqlc.GetOAuth2AccountIDParams{Issuer: issuer, Subject: subject})
		if err != nil || owner != account.ID {
			t.Fatal("canonical identity owner changed", err)
		}
		var exists bool
		if err := locker.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM parties WHERE id=$1) OR EXISTS(SELECT 1 FROM persons WHERE party_id=$1) OR EXISTS(SELECT 1 FROM accounts WHERE id=$2)", losingPerson, loser.ID).Scan(&exists); err != nil || exists {
			t.Fatal("losing registration fixture survived rollback", err)
		}
		t.Log("PASS: one canonical identity owner, losing account creation rolled back")
	})
}
