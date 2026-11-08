package routers

import (
	"fmt"
	"net/http"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
)

func TestOrganizationOwnerLifecycleHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	ownerID, ownerToken := organizationTestVerifiedAccount(t, ctx, pool)
	candidateID, candidateToken := organizationTestVerifiedAccount(t, ctx, pool)
	_, outsiderToken := organizationTestVerifiedAccount(t, ctx, pool)
	owner, err := db.NewStore(pool).GetAccountByID(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := db.NewStore(pool).GetAccountByID(ctx, candidateID)
	if err != nil {
		t.Fatal(err)
	}
	status, created := authAcceptanceCall(t, server, "/organizations", http.MethodPost,
		map[string]string{"name": "Owner transfer test"}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("create organization: status=%d response=%v", status, created)
	}
	orgID := int64(created["party_id"].(float64))
	memberURL := fmt.Sprintf("/organizations/%d/members", orgID)
	status, _ = authAcceptanceCall(t, server, memberURL, http.MethodPost, map[string]int64{"person_id": candidate.PersonID}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("add ordinary member status=%d", status)
	}
	promoteURL := fmt.Sprintf("/organizations/%d/owners/%d", orgID, candidate.PersonID)
	demoteURL := fmt.Sprintf("/organizations/%d/owners/%d", orgID, owner.PersonID)
	transferURL := fmt.Sprintf("/organizations/%d/owners/transfer", orgID)
	status, _ = authAcceptanceCall(t, server, promoteURL, http.MethodPut, nil, outsiderToken)
	if status != http.StatusForbidden {
		t.Fatalf("outsider promoted owner: %d", status)
	}
	status, _ = authAcceptanceCall(t, server, demoteURL, http.MethodDelete, nil, ownerToken)
	if status != http.StatusConflict {
		t.Fatalf("last owner demoted status=%d", status)
	}
	status, member := authAcceptanceCall(t, server, promoteURL, http.MethodPut, nil, ownerToken)
	if status != http.StatusOK || member["role"] != "owner" {
		t.Fatalf("promotion status=%d result=%v", status, member)
	}
	status, _ = authAcceptanceCall(t, server, demoteURL, http.MethodDelete, nil, ownerToken)
	if status != http.StatusNoContent {
		t.Fatalf("demote owner after promotion: %d", status)
	}
	status, _ = authAcceptanceCall(t, server, transferURL, http.MethodPost,
		map[string]int64{"person_id": owner.PersonID}, ownerToken)
	if status != http.StatusForbidden {
		t.Fatalf("former owner could transfer: %d", status)
	}
	status, reassigned := authAcceptanceCall(t, server, transferURL, http.MethodPost,
		map[string]int64{"person_id": owner.PersonID}, candidateToken)
	if status != http.StatusOK || reassigned["role"] != "owner" {
		t.Fatalf("transfer status=%d result=%v", status, reassigned)
	}
	status, _ = authAcceptanceCall(t, server, demoteURL, http.MethodDelete, nil, ownerToken)
	if status != http.StatusConflict {
		t.Fatalf("new sole owner demoted: %d", status)
	}
	status, view := organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d", orgID), ownerToken)
	if status != http.StatusOK {
		t.Fatalf("owner no longer sees organization: %d %v", status, view)
	}
	a := view.(map[string]any)
	if a["membership"].(map[string]any)["role"] != "owner" {
		t.Fatalf("owner role not projected: %v", a)
	}
	status, other := organizationGetJSON(t, server, fmt.Sprintf("/organizations/%d", orgID), candidateToken)
	if status != http.StatusOK || other.(map[string]any)["membership"].(map[string]any)["role"] != "member" {
		t.Fatalf("transferring actor still owner: status=%d result=%v", status, other)
	}
}
