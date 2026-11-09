package routers

import (
	"fmt"
	"net/http"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/google/uuid"
)

// Exercises the actual Gin authentication chain, SQL authorization,
// outbox write, and single-use invitation redemption over HTTP.
func TestTeamInvitationAuthorizationAndReplayHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	store := db.NewStore(pool)
	_, ownerBearer := organizationTestVerifiedAccount(t, ctx, pool)
	recipientID, recipientBearer := organizationTestVerifiedAccount(t, ctx, pool)
	_, outsiderBearer := organizationTestVerifiedAccount(t, ctx, pool)
	recipient, err := store.GetAccountByID(ctx, recipientID)
	if err != nil {
		t.Fatal(err)
	}

	status, org := authAcceptanceCall(t, server, "/orgs", http.MethodPost,
		map[string]string{"name": "Invitation HTTP " + uuid.NewString()}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("organization creation: %d %v", status, org)
	}
	orgID := int64(org["party_id"].(float64))
	status, added := authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/members", orgID), http.MethodPost,
		map[string]int64{"person_id": recipient.PersonID}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("add organization member: %d %v", status, added)
	}
	status, team := authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/teams", orgID), http.MethodPost,
		map[string]string{"name": "Backend"}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("create team: %d %v", status, team)
	}
	teamID := int64(team["id"].(float64))
	listStatus, listed := organizationGetJSON(t, server,
		fmt.Sprintf("/orgs/%d/teams", orgID), ownerBearer)
	if listStatus != http.StatusOK || len(listed.([]any)) != 1 ||
		int64(listed.([]any)[0].(map[string]any)["id"].(float64)) != teamID {
		t.Fatalf("owner team listing changed: status=%d teams=%v", listStatus, listed)
	}
	emptyStatus, empty := organizationGetJSON(t, server,
		fmt.Sprintf("/teams/%d/members", teamID), ownerBearer)
	if emptyStatus != http.StatusOK || len(empty.([]any)) != 0 {
		t.Fatalf("authorized empty membership list: status=%d members=%v", emptyStatus, empty)
	}
	emptyStatus, empty = organizationGetJSON(t, server,
		fmt.Sprintf("/orgs/%d/teams?page=9999", orgID), ownerBearer)
	if emptyStatus != http.StatusOK || len(empty.([]any)) != 0 {
		t.Fatalf("authorized empty teams page: status=%d teams=%v", emptyStatus, empty)
	}
	emptyStatus, _ = organizationGetJSON(t, server,
		fmt.Sprintf("/orgs/%d/teams", orgID), outsiderBearer)
	if emptyStatus != http.StatusNotFound {
		t.Fatalf("nonmember must not list organization teams: status=%d", emptyStatus)
	}
	// Visibility assignment now precedes action-level pagination. An outsider
	// cannot distinguish a valid query from an invalid one on a hidden org.
	emptyStatus, _ = organizationGetJSON(t, server,
		fmt.Sprintf("/orgs/%d/teams?page=invalid", orgID), outsiderBearer)
	if emptyStatus != http.StatusNotFound {
		t.Fatalf("invisible org with bad pagination must remain hidden: status=%d", emptyStatus)
	}
	invitePath := fmt.Sprintf("/teams/%d/invites", teamID)

	status, denied := authAcceptanceCall(t, server, invitePath, http.MethodPost,
		map[string]string{"email": recipient.Email}, recipientBearer)
	if status != http.StatusForbidden || denied["code"] != "organization.team_owner_required" {
		t.Fatalf("non-owner issued invitation: %d %v", status, denied)
	}
	status, issued := authAcceptanceCall(t, server, invitePath, http.MethodPost,
		map[string]string{"email": recipient.Email}, ownerBearer)
	if status != http.StatusAccepted {
		t.Fatalf("issue invitation: %d %v", status, issued)
	}
	if _, exposed := issued["token"]; exposed {
		t.Fatal("public invitation response exposed raw bearer token")
	}
	id, err := uuid.Parse(issued["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	var intents int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM job_outbox
		WHERE task_type='task:send_team_invite'
		  AND payload->>'invite_id'=$1
	`, id.String()).Scan(&intents); err != nil || intents != 1 {
		t.Fatalf("invitation was not durably queued: intents=%d err=%v", intents, err)
	}
	token, err := secret.DeriveToken(server.config.TokenSymmetricKey, "team-invite:v1", id[:])
	if err != nil {
		t.Fatal(err)
	}
	// A valid-looking bearer proof does not bypass verified email ownership.
	status, denied = authAcceptanceCall(t, server, "/teams/invites/accept", http.MethodPost,
		map[string]string{"token": token}, outsiderBearer)
	if status != http.StatusBadRequest || denied["code"] != "organization.team_invite_invalid" {
		t.Fatalf("wrong recipient accepted proof: %d %v", status, denied)
	}
	status, accepted := authAcceptanceCall(t, server, "/teams/invites/accept", http.MethodPost,
		map[string]string{"token": token}, recipientBearer)
	if status != http.StatusCreated || int64(accepted["person_id"].(float64)) != recipient.PersonID {
		t.Fatalf("verified recipient could not redeem: %d %v", status, accepted)
	}
	status, replay := authAcceptanceCall(t, server, "/teams/invites/accept", http.MethodPost,
		map[string]string{"token": token}, recipientBearer)
	if status != http.StatusBadRequest || replay["code"] != "organization.team_invite_invalid" {
		t.Fatalf("consumed proof replayed: %d %v", status, replay)
	}
	status, members := organizationGetJSON(t, server,
		fmt.Sprintf("/teams/%d/members?page=1&limit=20", teamID), recipientBearer)
	if status != http.StatusOK || len(members.([]any)) != 1 {
		t.Fatalf("team membership not readable: %d %v", status, members)
	}
	status, outsiders := organizationGetJSON(t, server,
		fmt.Sprintf("/teams/%d/members?page=1&limit=20", teamID), outsiderBearer)
	if status != http.StatusNotFound {
		t.Fatalf("outsider must not see private team membership endpoint: %d %v", status, outsiders)
	}
	// A JWT principal never grants ownership by itself; database membership
	// controls all team operations throughout the request.
}

func TestTeamInvitationRevokeHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	store := db.NewStore(pool)
	_, ownerBearer := organizationTestVerifiedAccount(t, ctx, pool)
	recipientID, recipientBearer := organizationTestVerifiedAccount(t, ctx, pool)
	recipient, err := store.GetAccountByID(ctx, recipientID)
	if err != nil {
		t.Fatal(err)
	}
	status, org := authAcceptanceCall(t, server, "/orgs", http.MethodPost,
		map[string]string{"name": "Revocation " + uuid.NewString()}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("organization creation: %d %v", status, org)
	}
	orgID := int64(org["party_id"].(float64))
	status, _ = authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/members", orgID), http.MethodPost,
		map[string]int64{"person_id": recipient.PersonID}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("add member: %d", status)
	}
	status, team := authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/teams", orgID), http.MethodPost,
		map[string]string{"name": "Review"}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("create team: %d %v", status, team)
	}
	teamID := int64(team["id"].(float64))
	status, invite := authAcceptanceCall(t, server, fmt.Sprintf("/teams/%d/invites", teamID), http.MethodPost,
		map[string]string{"email": recipient.Email}, ownerBearer)
	if status != http.StatusAccepted {
		t.Fatalf("create invite: %d %v", status, invite)
	}
	id, err := uuid.Parse(invite["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	token, err := secret.DeriveToken(server.config.TokenSymmetricKey, "team-invite:v1", id[:])
	if err != nil {
		t.Fatal(err)
	}
	revokePath := fmt.Sprintf("/teams/%d/invites/%s", teamID, id)
	status, rejected := authAcceptanceCall(t, server, revokePath, http.MethodDelete, nil, recipientBearer)
	if status != http.StatusBadRequest && status != http.StatusForbidden {
		t.Fatalf("non-owner revoked invitation: %d %v", status, rejected)
	}
	status, _ = authAcceptanceCall(t, server, revokePath, http.MethodDelete, nil, ownerBearer)
	if status != http.StatusNoContent {
		t.Fatalf("owner could not revoke: %d", status)
	}
	status, rejected = authAcceptanceCall(t, server, "/teams/invites/accept", http.MethodPost,
		map[string]string{"token": token}, recipientBearer)
	if status != http.StatusBadRequest || rejected["code"] != "organization.team_invite_invalid" {
		t.Fatalf("revoked invitation accepted: %d %v", status, rejected)
	}
}

func TestTeamMembershipActionsPreserveOwnerAndSQLCContracts(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	_, ownerBearer := organizationTestVerifiedAccount(t, ctx, pool)
	recipientID, recipientBearer := organizationTestVerifiedAccount(t, ctx, pool)
	_, outsiderBearer := organizationTestVerifiedAccount(t, ctx, pool)
	store := db.NewStore(pool)
	recipient, err := store.GetAccountByID(ctx, recipientID)
	if err != nil {
		t.Fatal(err)
	}
	status, org := authAcceptanceCall(t, server, "/orgs", http.MethodPost,
		map[string]string{"name": "Membership HTTP " + uuid.NewString()}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("create org: %d %v", status, org)
	}
	orgID := int64(org["party_id"].(float64))
	status, _ = authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/members", orgID),
		http.MethodPost, map[string]int64{"person_id": recipient.PersonID}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("recipient must join Organization: %d", status)
	}
	status, team := authAcceptanceCall(t, server, fmt.Sprintf("/orgs/%d/teams", orgID),
		http.MethodPost, map[string]string{"name": "Backend"}, ownerBearer)
	if status != http.StatusCreated {
		t.Fatalf("create team: %d %v", status, team)
	}
	teamID := int64(team["id"].(float64))
	path := fmt.Sprintf("/teams/%d/members", teamID)
	add := map[string]int64{"person_id": recipient.PersonID}
	status, denied := authAcceptanceCall(t, server, path, http.MethodPost, add, recipientBearer)
	if status != http.StatusForbidden || denied["code"] != "organization.team_owner_required" {
		t.Fatalf("non-owner created membership: %d %v", status, denied)
	}
	status, created := authAcceptanceCall(t, server, path, http.MethodPost, add, ownerBearer)
	if status != http.StatusCreated || int64(created["person_id"].(float64)) != recipient.PersonID {
		t.Fatalf("owner could not add member: %d %v", status, created)
	}
	status, duplicate := authAcceptanceCall(t, server, path, http.MethodPost, add, ownerBearer)
	if status != http.StatusConflict || duplicate["code"] != "organization.team_member_exists" {
		t.Fatalf("duplicate insertion not rejected: %d %v", status, duplicate)
	}
	status, members := organizationGetJSON(t, server, path+"?page=1&limit=20", recipientBearer)
	if status != http.StatusOK || len(members.([]any)) != 1 {
		t.Fatalf("member cannot read Team: %d %v", status, members)
	}
	status, _ = organizationGetJSON(t, server, path, outsiderBearer)
	if status != http.StatusNotFound {
		t.Fatalf("outsider leaked Team member data: %d", status)
	}
	deletePath := fmt.Sprintf("%s/%d", path, recipient.PersonID)
	status, denied = authAcceptanceCall(t, server, deletePath, http.MethodDelete, nil, recipientBearer)
	if status != http.StatusForbidden || denied["code"] != "organization.team_owner_required" {
		t.Fatalf("nonowner removed member: %d %v", status, denied)
	}
	status, _ = authAcceptanceCall(t, server, deletePath, http.MethodDelete, nil, ownerBearer)
	if status != http.StatusNoContent {
		t.Fatalf("owner could not remove member: %d", status)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM team_members WHERE team_id=$1 AND person_id=$2",
		teamID, recipient.PersonID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("membership survived deletion: count=%d err=%v", count, err)
	}
	status, _ = authAcceptanceCall(t, server, deletePath, http.MethodDelete, nil, ownerBearer)
	if status != http.StatusNotFound {
		t.Fatalf("duplicate deletion must fail: %d", status)
	}
}
