package routers

import (
	"fmt"
	"net/http"
	"testing"

	account_model "github.com/datvtph41107/bdspro/models/account"
)

func TestOrganizationRemoveOrdinaryMemberHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	ownerID, ownerToken := organizationTestVerifiedAccount(t, ctx, pool)
	memberID, memberToken := organizationTestVerifiedAccount(t, ctx, pool)
	_, outsiderToken := organizationTestVerifiedAccount(t, ctx, pool)
	ownerAccount, err := account_model.ReadByID(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	memberAccount, err := account_model.ReadByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	status, created := authAcceptanceCall(t, server, "/orgs", http.MethodPost,
		map[string]string{"name": "Removal HTTP Contract"}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("create organization %d %v", status, created)
	}
	orgID := int64(created["party_id"].(float64))
	memberPath := fmt.Sprintf("/orgs/%d/members/%d", orgID, memberAccount.PersonID)
	ownerPath := fmt.Sprintf("/orgs/%d/members/%d", orgID, ownerAccount.PersonID)
	addPath := fmt.Sprintf("/orgs/%d/members", orgID)
	status, _ = authAcceptanceCall(t, server, addPath, http.MethodPost,
		map[string]int64{"person_id": memberAccount.PersonID}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("add member status=%d", status)
	}
	status, _ = authAcceptanceCall(t, server, ownerPath, http.MethodDelete, nil, ownerToken)
	if status != http.StatusNotFound {
		t.Fatalf("owner role unexpectedly removable: status=%d", status)
	}
	status, _ = authAcceptanceCall(t, server, memberPath, http.MethodDelete, nil, outsiderToken)
	if status != http.StatusNotFound {
		t.Fatalf("outsider learned or modified member state: status=%d", status)
	}
	status, _ = authAcceptanceCall(t, server, memberPath, http.MethodDelete, nil, ownerToken)
	if status != http.StatusNoContent {
		t.Fatalf("owner revocation status=%d", status)
	}
	status, _ = authAcceptanceCall(t, server, memberPath, http.MethodDelete, nil, ownerToken)
	if status != http.StatusNotFound {
		t.Fatalf("replayed revocation status=%d", status)
	}
	status, _ = organizationGetJSON(t, server, fmt.Sprintf("/orgs/%d", orgID), memberToken)
	if status != http.StatusNotFound {
		t.Fatalf("ended member still has organization view: status=%d", status)
	}
	status, members := organizationGetJSON(t, server, addPath, ownerToken)
	if status != http.StatusOK {
		t.Fatalf("owner lost access to membership view: %d %v", status, members)
	}
	if len(members.([]any)) != 1 {
		t.Fatalf("owner membership invariant lost: %v", members)
	}
}
