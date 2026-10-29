package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func organizationTestVerifiedAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (int64, string) {
    t.Helper()
    account, err := db.NewStore(pool).CreateGoogleAccountTx(ctx,
        uuid.NewString()+"@example.test", uuid.NewString())
    if err != nil { t.Fatal(err) }
    if !account.IsEmailVerified { t.Fatal("test account must have verified email") }
    maker, err := token.NewJWTMaker(strings.Repeat("x", 32))
    if err != nil { t.Fatal(err) }
    access, _, err := maker.CreateToken(account.ID, time.Minute, token.TokenTypeAccess)
    if err != nil { t.Fatal(err) }
    return account.ID, access
}

func organizationGetJSON(t *testing.T, server *Server, path, bearer string) (int, any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	var result any
	if w.Body.Len() != 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: status=%d body=%s: %v", w.Code, w.Body.String(), err)
		}
	}
	return w.Code, result
}

func accessPermissionSet(value any) map[string]bool {
	permissions := make(map[string]bool)
	for _, item := range value.([]any) {
		permissions[item.(string)] = true
	}
	return permissions
}

func TestOrganizationAccountMembershipRoleAndAuthorizationHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	ownerID, ownerToken := organizationTestVerifiedAccount(t, ctx, pool)
	memberID, memberToken := organizationTestVerifiedAccount(t, ctx, pool)
	_, outsiderToken := organizationTestVerifiedAccount(t, ctx, pool)

	status, org := authAcceptanceCall(t, server, "/organizations", http.MethodPost, map[string]string{"name": "Company A"}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("organization creation status=%d result=%v", status, org)
	}
	orgID := int64(org["party_id"].(float64))
	status, second := authAcceptanceCall(t, server, "/organizations", http.MethodPost, map[string]string{"name": "Company B"}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("second organization creation %d %v", status, second)
	}

	memberAccount, err := db.NewStore(pool).GetAccountByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	status, added := authAcceptanceCall(t, server, fmt.Sprintf("/organizations/%d/members", orgID), http.MethodPost, map[string]int64{"person_id": memberAccount.PersonID}, ownerToken)
	if status != http.StatusCreated || added["role"] != "member" {
		t.Fatalf("add member status=%d result=%v", status, added)
	}

	status, ownerList := organizationGetJSON(t, server, "/organizations", ownerToken)
	if status != http.StatusOK {
		t.Fatalf("list owner status=%d", status)
	}
	ownerOrganizations := ownerList.([]any)
	if len(ownerOrganizations) != 2 {
		t.Fatalf("owner must see two organizations: %v", ownerList)
	}
	for _, v := range ownerOrganizations {
		o := v.(map[string]any)
		if o["membership"].(map[string]any)["role"] != "owner" {
			t.Fatalf("wrong owner role: %v", o)
		}
		p := accessPermissionSet(o["permissions"])
		if !p["organization.view"] || !p["organization.update"] || !p["organization.members.add"] || !p["organization.members.view"] {
			t.Fatalf("owner permissions missing: %v", p)
		}
	}

	status, memberList := organizationGetJSON(t, server, "/organizations", memberToken)
	if status != http.StatusOK {
		t.Fatalf("list member status=%d", status)
	}
	memberOrgs := memberList.([]any)
	if len(memberOrgs) != 1 {
		t.Fatalf("member must see exactly one organization: %v", memberList)
	}
	memberOrg := memberOrgs[0].(map[string]any)
	if int64(memberOrg["party_id"].(float64)) != orgID || memberOrg["membership"].(map[string]any)["role"] != "member" {
		t.Fatalf("membership/role mismatch: %v", memberOrg)
	}
	memberPerms := accessPermissionSet(memberOrg["permissions"])
	if len(memberPerms) != 1 || !memberPerms["organization.view"] {
		t.Fatalf("member must only have view: %v", memberPerms)
	}

	status, detail := organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d", orgID), memberToken)
	if status != http.StatusOK || detail.(map[string]any)["membership"].(map[string]any)["role"] != "member" {
		t.Fatalf("member detail status=%d result=%v", status, detail)
	}
	status, other := organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d", orgID), outsiderToken)
	if status != http.StatusNotFound {
		t.Fatalf("outsider leaked organization: %d %v", status, other)
	}
	status, none := organizationGetJSON(t, server, "/organizations", outsiderToken)
	if status != http.StatusOK || len(none.([]any)) != 0 {
		t.Fatalf("outsider list status=%d result=%v", status, none)
	}

	status, ownerMembers := organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d/members", orgID), ownerToken)
	if status != http.StatusOK || len(ownerMembers.([]any)) != 2 {
		t.Fatalf("owner member list status=%d members=%v", status, ownerMembers)
	}
	for _, v := range ownerMembers.([]any) {
		m := v.(map[string]any)
		if _, ok := m["email"]; ok {
			t.Fatal("membership response exposed email")
		}
		if _, ok := m["phone"]; ok {
			t.Fatal("membership response exposed phone")
		}
	}
	for _, token := range []string{memberToken, outsiderToken} {
		status, _ = organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d/members", orgID), token)
		if status != http.StatusNotFound {
			t.Fatalf("non-owner read membership: %d", status)
		}
	}
	status, _ = authAcceptanceCall(t, server, fmt.Sprintf("/organizations/%d", orgID), http.MethodPatch, map[string]string{"name": "Unauthorized"}, memberToken)
	if status != http.StatusForbidden {
		t.Fatalf("member mutated organization: %d", status)
	}

	// Losing active membership immediately removes read rights; no JWT reissue
	// is needed because all reads evaluate the current database membership.
	if _, err := pool.Exec(ctx, "UPDATE organization_memberships SET ended_at=now() WHERE organization_id=$1 AND person_id=$2", orgID, memberAccount.PersonID); err != nil {
		t.Fatal(err)
	}
	status, after := organizationGetJSON(t, server, "/organizations", memberToken)
	if status != http.StatusOK || len(after.([]any)) != 0 {
		t.Fatalf("ended membership still visible: %d %v", status, after)
	}
	status, _ = organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d", orgID), memberToken)
	if status != http.StatusNotFound {
		t.Fatalf("ended membership can read detail: %d", status)
	}
	if ownerID <= 0 {
		t.Fatal("invalid owner")
	}
}
