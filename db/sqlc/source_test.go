package db

import (
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestSourceQueryLifecycle(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	orgID := createRandomOrganization(t, ctx, q)
	personID, accountID, _ := createVerifiedAccount(t, ctx, q)

	memberID, err := q.UpsertInitialOrganizationMembership(ctx, UpsertInitialOrganizationMembershipParams{
		OrganizationID: orgID, PersonID: personID,
	})
	require.NoError(t, err)
	roleID, err := q.CreateWorkspaceAdminRole(ctx, orgID)
	require.NoError(t, err)
	require.NoError(t, q.GrantInitialRolePermissions(ctx, roleID))
	require.NoError(t, q.CreateInitialRoleAssignment(ctx, CreateInitialRoleAssignmentParams{
		OrganizationID: orgID, MembershipID: memberID, RoleID: roleID,
	}))

	requestKey := "capture-" + util.RandomString(12)
	requestHash := strings.Repeat("a", 64)
	capture := InsertSourceIntakeParams{
		ActorAccountID:  accountID,
		OrganizationID:  orgID,
		Channel:         "zalo",
		Label:           "house",
		AreaM2:          "75",
		AreaBasis:       "",
		ReceivedAt:      "",
		Summary:         "Lecture 05 test source",
		RequestKey:      requestKey,
		RequestHash:     requestHash,
		HasInitialPrice: true,
	}

	intakeID, err := q.InsertSourceIntake(ctx, capture)
	require.NoError(t, err)
	require.Positive(t, intakeID)

	read, err := q.GetAuthorizedSource(ctx, GetAuthorizedSourceParams{
		ActorAccountID: accountID, OrganizationID: orgID, IntakeID: intakeID,
	})
	require.NoError(t, err)
	require.Equal(t, intakeID, read.ID)
	require.Equal(t, orgID, read.OrganizationID)
	require.Equal(t, "zalo", read.SourceChannelCode)
	require.Contains(t, read.ReportedAreaM2, "75")

	replayed, err := q.FindAuthorizedCaptureReplay(ctx, FindAuthorizedCaptureReplayParams{
		ActorAccountID: accountID, OrganizationID: orgID, RequestKey: requestKey,
	})
	require.NoError(t, err)
	require.Equal(t, intakeID, replayed.ID)
	require.Equal(t, requestHash, replayed.RequestHash)

	// ON CONFLICT DO NOTHING RETURNING returns ErrNoRows for a duplicate key.
	_, err = q.InsertSourceIntake(ctx, capture)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	priceKey := "price-" + util.RandomString(12)
	priceHash := strings.Repeat("b", 64)
	price := InsertPriceReportParams{
		ActorAccountID:   accountID,
		OrganizationID:   orgID,
		IntakeID:         intakeID,
		Amount:           "4200000000",
		Currency:         "VND",
		PriceBasis:       "",
		SourceBasis:      "",
		StatementAt:      "",
		Note:             "test quotation",
		PriceRequestKey:  priceKey,
		PriceRequestHash: priceHash,
	}
	priceID, err := q.InsertPriceReport(ctx, price)
	require.NoError(t, err)
	require.Positive(t, priceID)

	history, err := q.ListAuthorizedPriceReports(ctx, ListAuthorizedPriceReportsParams{
		ActorAccountID: accountID, OrganizationID: orgID, IntakeID: intakeID,
	})
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, priceID, history[0].ID)
	require.Equal(t, "VND", history[0].CurrencyCode)
	require.Contains(t, history[0].Amount, "4200000000")

	priceReplay, err := q.FindAuthorizedPriceReplay(ctx, FindAuthorizedPriceReplayParams{
		ActorAccountID: accountID, OrganizationID: orgID,
		IntakeID: intakeID, PriceRequestKey: priceKey,
	})
	require.NoError(t, err)
	require.Equal(t, priceID, priceReplay.ID)
	require.Equal(t, priceHash, priceReplay.RequestHash)

	_, err = q.InsertPriceReport(ctx, price)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Account without tenant membership must never read another tenant's source.
	_, strangerID := createRandomAccount(t, ctx, q)
	_, err = q.GetAuthorizedSource(ctx, GetAuthorizedSourceParams{
		ActorAccountID: strangerID, OrganizationID: orgID, IntakeID: intakeID,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)

	history, err = q.ListAuthorizedPriceReports(ctx, ListAuthorizedPriceReportsParams{
		ActorAccountID: strangerID, OrganizationID: orgID, IntakeID: intakeID,
	})
	require.NoError(t, err)
	require.Empty(t, history)
}

func TestSourceCreateRequiresMembership(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	orgID := createRandomOrganization(t, ctx, q)
	_, strangerAccountID := createRandomAccount(t, ctx, q)

	_, err := q.InsertSourceIntake(ctx, InsertSourceIntakeParams{
		ActorAccountID: strangerAccountID,
		OrganizationID: orgID,
		Channel:        "zalo",
		RequestKey:     "forbidden-" + util.RandomString(12),
		RequestHash:    strings.Repeat("c", 64),
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
