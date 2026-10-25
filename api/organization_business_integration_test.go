package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// A real business vertical slice: HTTP/JWT actor -> handler -> SQL ownership
// predicate -> durable state. It writes ONLY into CI disposable PostgreSQL.
// Membership expiration is set directly by the test because BDSPro does not
// currently expose an API for ending an organization membership.
func TestBusinessOrganizationActorLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("BUSINESS_TEST_DB_URL")
	if os.Getenv("GITHUB_ACTIONS") != "true" || dsn == "" {
		t.Skip("organization business integration requires CI disposable PostgreSQL")
	}
	target, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", target.Hostname(), "must not reach a remote database")
	require.Equal(t, "5432", target.Port(), "must use the disposable CI postgres service")
	require.Equal(t, "/bdspro", target.Path)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var name string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&name))
	require.Equal(t, "bdspro", name)
	store := db.NewStore(pool)

	createAccount := func() db.Account {
		t.Helper()
		account, err := store.RegisterTx(ctx, db.RegisterTxParams{
			Email: uuid.NewString() + "@example.test", PasswordHash: "test-hash",
		})
		require.NoError(t, err)
		return account
	}
	owner := createAccount()
	member := createAccount()
	otherOwner := createAccount()
	app := newTestServer(t, store)

	call := func(actorID int64, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if actorID != 0 {
			jwt, _, err := app.tokenMaker.CreateToken(actorID, time.Hour, token.TokenTypeAccess)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+jwt)
		}
		res := httptest.NewRecorder()
		app.Handler().ServeHTTP(res, req)
		return res
	}
	organizationsFor := func(account db.Account) []db.Organization {
		t.Helper()
		res := call(account.ID, http.MethodGet, "/organizations", "")
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		var organizations []db.Organization
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &organizations))
		return organizations
	}
	createOrganization := func(account db.Account, label string) db.Organization {
		t.Helper()
		res := call(account.ID, http.MethodPost, "/organizations",
			fmt.Sprintf(`{"name":%q}`, label))
		require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
		var organization db.Organization
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &organization))
		require.Positive(t, organization.PartyID)
		require.Equal(t, label, organization.Name)
		var activeOwnerCount int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM organization_memberships
			WHERE organization_id=$1 AND person_id=$2
			  AND role='owner' AND ended_at IS NULL`,
			organization.PartyID, account.PersonID).Scan(&activeOwnerCount))
		require.Equal(t, 1, activeOwnerCount, "organization creation must persist its owner")
		return organization
	}

	// Being unauthenticated is different from being authenticated as a
	// non-owner. Neither may create a resource on behalf of another actor.
	require.Equal(t, http.StatusUnauthorized,
		call(0, http.MethodPost, "/organizations", `{"name":"denied"}`).Code)

	owned := createOrganization(owner, "Organization A")
	other := createOrganization(otherOwner, "Organization B")
	ownedPath := fmt.Sprintf("/organizations/%d", owned.PartyID)
	otherPath := fmt.Sprintf("/organizations/%d", other.PartyID)
	memberPath := ownedPath + "/members"
	addMemberBody := fmt.Sprintf(`{"person_id":%d}`, member.PersonID)

	// A valid access JWT for B does not authorize writes to A's resource.
	require.Equal(t, http.StatusForbidden,
		call(otherOwner.ID, http.MethodPatch, ownedPath, `{"name":"stolen"}`).Code)
	require.Equal(t, http.StatusForbidden,
		call(member.ID, http.MethodPost, memberPath, addMemberBody).Code)
	require.Len(t, organizationsFor(owner), 1)
	require.Len(t, organizationsFor(otherOwner), 1)
	require.Empty(t, organizationsFor(member))

	// Owner grants membership: B can list A's organization but cannot
	// acquire the owner's mutation rights through its member role.
	res := call(owner.ID, http.MethodPost, memberPath, addMemberBody)
	require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
	require.Equal(t, http.StatusConflict,
		call(owner.ID, http.MethodPost, memberPath, addMemberBody).Code,
		"active membership may not be duplicated")
	require.Equal(t, http.StatusForbidden,
		call(member.ID, http.MethodPatch, ownedPath, `{"name":"member-renamed"}`).Code)
	require.Equal(t, owned.PartyID, organizationsFor(member)[0].PartyID)
	require.Equal(t, other.PartyID, organizationsFor(otherOwner)[0].PartyID)

	// The legitimate owner changes only their organization, not the other.
	require.Equal(t, http.StatusOK,
		call(owner.ID, http.MethodPatch, ownedPath, `{"name":"Organization A Updated"}`).Code)
	require.Equal(t, http.StatusOK,
		call(otherOwner.ID, http.MethodPatch, otherPath, `{"name":"Organization B Updated"}`).Code)

	// Membership expiry is a test-only SQL state fixture, not a public API.
	// It must immediately remove visibility for that Person.
	tag, err := pool.Exec(ctx, `
		UPDATE organization_memberships SET ended_at=now()
		WHERE organization_id=$1 AND person_id=$2
		  AND role='member' AND ended_at IS NULL`,
		owned.PartyID, member.PersonID)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
	require.Empty(t, organizationsFor(member))
	require.Equal(t, owned.PartyID, organizationsFor(owner)[0].PartyID)

	// Once the owner's membership ends, a previously valid JWT must not
	// authorize further changes: current SQL membership state wins.
	tag, err = pool.Exec(ctx, `
		UPDATE organization_memberships SET ended_at=now()
		WHERE organization_id=$1 AND person_id=$2
		  AND role='owner' AND ended_at IS NULL`,
		owned.PartyID, owner.PersonID)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
	require.Equal(t, http.StatusForbidden,
		call(owner.ID, http.MethodPatch, ownedPath, `{"name":"must-not-change"}`).Code)
	require.Equal(t, http.StatusForbidden,
		call(owner.ID, http.MethodPost, memberPath, addMemberBody).Code)
	require.Empty(t, organizationsFor(owner))

	// Persisted data, not just HTTP status, confirms unauthorized writes
	// never changed either organization or inserted a new membership.
	var currentName string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT name FROM organizations WHERE party_id=$1", owned.PartyID).Scan(&currentName))
	require.Equal(t, "Organization A Updated", currentName)
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT name FROM organizations WHERE party_id=$1", other.PartyID).Scan(&currentName))
	require.Equal(t, "Organization B Updated", currentName)
	var activeMembershipCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM organization_memberships
		WHERE organization_id=$1 AND ended_at IS NULL`, owned.PartyID).
		Scan(&activeMembershipCount))
	require.Zero(t, activeMembershipCount)
}
