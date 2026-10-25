package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// These tests are an executable reference for ENG-CORE-09. They exercise
// actual registered routes and middleware, not a parallel test router.
// An unconfigured Store means an accidental persistence call cannot pass.
func TestFeatureDeliveryProtectedRoutesRejectBeforeStore(t *testing.T) {
	routes := []struct {
		name, method, path, body string
	}{
		{"Me", http.MethodGet, "/users/me", ""},
		{"CreateOrganization", http.MethodPost, "/organizations", `{"name":"Company"}`},
		{"ListOrganizations", http.MethodGet, "/organizations", ""},
		{"AddMember", http.MethodPost, "/organizations/20/members", `{"person_id":17}`},
		{"UpdateOrganization", http.MethodPatch, "/organizations/20", `{"name":"Company"}`},
	}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			server := newTestServer(t, nil)
			refresh, _, err := server.tokenMaker.CreateToken(10, time.Hour, token.TokenTypeRefresh)
			require.NoError(t, err)

			for _, tc := range []struct {
				name, bearer string
			}{
				{"MissingAccessToken", ""},
				{"WrongTokenPurpose", "Bearer " + refresh},
				{"MalformedAccessToken", "Bearer invalid-jwt"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
					req.Header.Set("Content-Type", "application/json")
					if tc.bearer != "" {
						req.Header.Set("Authorization", tc.bearer)
					}
					rec := httptest.NewRecorder()
					server.Handler().ServeHTTP(rec, req)
					require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
					var response map[string]string
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
					require.NotEmpty(t, response["error"])
					require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
				})
			}
		})
	}
}

// Input validation must reject bad requests before application/data operations.
// Source owner: api/organization.go. No mock expectations means any DB call fails.
func TestFeatureDeliveryOrganizationInvalidInputStopsBeforeStore(t *testing.T) {
	cases := []struct {
		name, method, path, body string
	}{
		{"CreateMissingName", http.MethodPost, "/organizations", `{}`},
		{"CreateWhitespaceName", http.MethodPost, "/organizations", `{"name":"   "}`},
		{"CreateMalformedJSON", http.MethodPost, "/organizations", `{"name":`},
		{"AddInvalidOrganizationID", http.MethodPost, "/organizations/0/members", `{"person_id":17}`},
		{"AddMissingPersonID", http.MethodPost, "/organizations/20/members", `{}`},
		{"AddNegativePersonID", http.MethodPost, "/organizations/20/members", `{"person_id":-1}`},
		{"UpdateMalformedOrganizationID", http.MethodPatch, "/organizations/no-id", `{"name":"Valid"}`},
		{"UpdateEmptyName", http.MethodPatch, "/organizations/20", `{"name":"  "}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(gomock.NewController(t))
			server := newTestServer(t, store)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, authRequest(t, server, tc.method, tc.path, tc.body))
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			var response map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.NotEmpty(t, response["error"])
		})
	}
}

type featureDeliveryContextKey struct{}

// Archetype A: multi-step command belongs in the Organization operation;
// the HTTP request supplies only a verified actor and normalized input.
func TestFeatureDeliveryCreateUsesOperationWithVerifiedActorAndContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), featureDeliveryContextKey{}, "request-owned")
	store := mockdb.NewMockStore(gomock.NewController(t))
	gomock.InOrder(
		store.EXPECT().GetAccountByID(gomock.Any(), int64(10)).
			DoAndReturn(func(actual context.Context, _ int64) (db.Account, error) {
				require.Equal(t, "request-owned", actual.Value(featureDeliveryContextKey{}))
				return db.Account{ID: 10, PersonID: 44}, nil
			}),
		store.EXPECT().CreateOrganizationTx(gomock.Any(), db.CreateOrganizationTxParams{
			Name: "BDSPro Example", OwnerPersonID: 44,
		}).DoAndReturn(func(actual context.Context, _ db.CreateOrganizationTxParams) (db.Organization, error) {
			require.Equal(t, "request-owned", actual.Value(featureDeliveryContextKey{}))
			return db.Organization{PartyID: 71, Name: "BDSPro Example"}, nil
		}),
	)
	server := newTestServer(t, store)
	// Client-provided actor or owner fields have no authority here.
	req := authRequest(t, server, http.MethodPost, "/organizations",
		`{"name":" BDSPro Example ","actor_account_id":999,"owner_person_id":999}`).WithContext(ctx)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var organization db.Organization
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &organization))
	require.Equal(t, int64(71), organization.PartyID)
	require.Equal(t, "BDSPro Example", organization.Name)
}

// Archetype B: a single SQL operation enforces resource authority. Do not
// add a generic service just to forward this call or trust a body actor ID.
func TestFeatureDeliveryMemberWriteKeepsAuthorityAtSQLBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		err error
		want int
	}{
		{"Owner", nil, http.StatusCreated},
		{"DeniedOwner", pgx.ErrNoRows, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), featureDeliveryContextKey{}, "request-owned")
			store := mockdb.NewMockStore(gomock.NewController(t))
			store.EXPECT().AddOrganizationMember(gomock.Any(), db.AddOrganizationMemberParams{
				OrganizationID: 20, PersonID: 17, ActorAccountID: 10,
			}).DoAndReturn(func(actual context.Context, arg db.AddOrganizationMemberParams) (db.OrganizationMembership, error) {
				require.Equal(t, "request-owned", actual.Value(featureDeliveryContextKey{}))
				if tc.err != nil {
					return db.OrganizationMembership{}, tc.err
				}
				return db.OrganizationMembership{
					ID: 61, OrganizationID: arg.OrganizationID, PersonID: arg.PersonID, Role: "member",
				}, nil
			})
			server := newTestServer(t, store)
			req := authRequest(t, server, http.MethodPost, "/organizations/20/members",
				`{"person_id":17,"actor_account_id":999,"role":"owner"}`).WithContext(ctx)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			require.Equal(t, tc.want, rec.Code, rec.Body.String())
			if tc.err == nil {
				var membership db.OrganizationMembership
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &membership))
				require.Equal(t, "member", membership.Role)
				require.Equal(t, int64(17), membership.PersonID)
			} else {
				require.JSONEq(t, `{"error":"organization owner required"}`, rec.Body.String())
			}
		})
	}
}
