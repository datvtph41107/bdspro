package db_test

import (
    "crypto/sha256"
    "errors"
    "testing"
    "time"

    account_model "github.com/datvtph41107/bdspro/models/account"
    auth_model "github.com/datvtph41107/bdspro/models/auth"
    db "github.com/datvtph41107/bdspro/models/db"
    "github.com/google/uuid"
)

// A pre-recovery consent preview cannot re-authorize the client, even when the
// caller later obtains a new first-party session. A new preview is required.
func TestRecoveryInvalidatesStaleConsentPreviewAndRequiresNewSession(t *testing.T) {
    ctx, _, store := authChannelsTestStore(t)
    oauth := store.(db.OAuthServerStore)

    owner, err := account_model.Register(ctx, store, account_model.RegisterParams{
        Email: uuid.NewString()+"@example.test", PasswordHash: "owner-hash",
    })
    if err != nil { t.Fatal(err) }

    fx := makeRecoveryFixture(t, ctx, store, oauth, owner.ID)
    staleVersion, err := oauth.GetOAuthConsentVersion(ctx, fx.accountID, fx.client)
    if err != nil { t.Fatal(err) }
    if staleVersion == 0 { t.Fatal("fixture did not create a consent epoch") }

    proof := resetRecoveryProof(t, ctx, store, fx.accountID)
    if err := auth_model.ResetPassword(ctx, store, proof, "recovered-hash"); err != nil {
        t.Fatal(err)
    }

    currentVersion, err := oauth.GetOAuthConsentVersion(ctx, fx.accountID, fx.client)
    if err != nil { t.Fatal(err) }
    if currentVersion <= staleVersion {
        t.Fatalf("recovery did not advance consent version: before=%d after=%d", staleVersion, currentVersion)
    }

    const redirect = "https://client.example.test/callback"
    const challenge = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ"
    codeOld := sha256.Sum256([]byte(uuid.NewString()))
    err = oauth.ApproveOAuthAuthCode(ctx, fx.client, fx.accountID, redirect,
        "profile:read", challenge, codeOld[:], time.Now().Add(time.Minute),
        staleVersion, fx.session)
    if !errors.Is(err, db.ErrOAuthSessionInactive) {
        t.Fatalf("old authorizing session survived recovery: %v", err)
    }

    // Simulate a *newly authenticated* first-party session, not a recovery
    // bypass. Session creation is separate from delegated OAuth consent.
    newSession, err := store.CreateSession(ctx, db.CreateSessionParams{
        ID: uuid.New(), AccountID: fx.accountID,
        RefreshToken: "post-recovery-" + uuid.NewString(),
        UserAgent: "post-recovery-consent",
        ClientIp: "127.0.0.1",
        ExpiresAt: time.Now().Add(time.Hour),
    })
    if err != nil { t.Fatal(err) }

    err = oauth.ApproveOAuthAuthCode(ctx, fx.client, fx.accountID, redirect,
        "profile:read", challenge, codeOld[:], time.Now().Add(time.Minute),
        staleVersion, newSession.ID)
    if !errors.Is(err, db.ErrOAuthConsentStale) {
        t.Fatalf("stale preview reauthorized client after recovery: %v", err)
    }

    codeNew := sha256.Sum256([]byte(uuid.NewString()))
    if err := oauth.ApproveOAuthAuthCode(ctx, fx.client, fx.accountID, redirect,
        "profile:read", challenge, codeNew[:], time.Now().Add(time.Minute),
        currentVersion, newSession.ID); err != nil {
        t.Fatalf("fresh authorization with current version failed: %v", err)
    }
    authorizations, err := oauth.ListOAuthAuthorizations(ctx, fx.accountID)
    if err != nil || len(authorizations) != 1 || authorizations[0].ClientID != fx.client {
        t.Fatalf("new consent was not recorded: %+v %v", authorizations, err)
    }
}

