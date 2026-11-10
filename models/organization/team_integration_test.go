package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

func TestTeamMembershipAndInvitationPostgres(t *testing.T) {
	ctx, store := integrationStore(t)
	owner := makeActor(t, ctx, store)
	invitee := makeActor(t, ctx, store)
	outsider := makeActor(t, ctx, store)

	org, err := organization.CreateOrganization(ctx, store, owner.ID, "Team test "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	team, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.ID, ActorAccountID: owner.ID, Name: "Engineering", Description: "Backend",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Empty visible resources remain 200 [] at the HTTP boundary; invisible
	// resources must return 404, even when the corresponding SELECT is empty.
	emptyMembers, err := organization.ListTeamUsers(ctx, store, owner.ID, team.ID,
		db.NewListOptions(1, 20, 20, 100))
	if err != nil || len(emptyMembers) != 0 {
		t.Fatalf("visible empty team members: count=%d err=%v", len(emptyMembers), err)
	}
	if _, err := organization.ListTeamUsers(ctx, store, outsider.ID, team.ID,
		db.NewListOptions(1, 20, 20, 100)); !errors.Is(err, organization.ErrTeamNotFound) {
		t.Fatalf("outsider list must hide team existence: %v", err)
	}
	if _, err := organization.ListTeamUsers(ctx, store, owner.ID, 9223372036854775807,
		db.NewListOptions(1, 20, 20, 100)); !errors.Is(err, organization.ErrTeamNotFound) {
		t.Fatalf("missing team list must return not found: %v", err)
	}
	// Out-of-range pages are legitimate empty lists for authorized callers.
	outOfRange, err := organization.ListTeams(ctx, store, owner.ID, org.ID,
		db.NewListOptions(99, 20, 20, 100))
	if err != nil || len(outOfRange) != 0 {
		t.Fatalf("authorized empty team page: count=%d err=%v", len(outOfRange), err)
	}
	emptyOrg, err := organization.CreateOrganization(ctx, store, owner.ID,
		"Empty team listing "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	emptyTeams, err := organization.ListTeams(ctx, store, owner.ID, emptyOrg.ID,
		db.NewListOptions(1, 20, 20, 100))
	if err != nil || len(emptyTeams) != 0 {
		t.Fatalf("valid empty organization teams: count=%d err=%v", len(emptyTeams), err)
	}
	if _, err := organization.ListTeams(ctx, store, outsider.ID, emptyOrg.ID,
		db.NewListOptions(1, 20, 20, 100)); !errors.Is(err, organization.ErrOrganizationNotFound) {
		t.Fatalf("outsider enumerated empty organization: %v", err)
	}
	if _, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.ID, ActorAccountID: owner.ID, Name: "engineering",
	}); !errors.Is(err, organization.ErrTeamAlreadyExists) {
		t.Fatalf("case-insensitive duplicate team accepted: %v", err)
	}
	if _, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.ID, ActorAccountID: outsider.ID, Name: "Unauthorized",
	}); !errors.Is(err, organization.ErrTeamOwnerRequired) {
		t.Fatalf("non-owner created a team: %v", err)
	}
	if _, err := organization.GetTeam(ctx, store, outsider.ID, team.ID); !errors.Is(err, organization.ErrTeamNotFound) {
		t.Fatalf("outsider read private team: %v", err)
	}
	if _, err := organization.RenameTeam(ctx, store, outsider.ID, team.ID, "Unauthorized rename"); !errors.Is(err, organization.ErrTeamOwnerRequired) {
		t.Fatalf("non-owner renamed a team: %v", err)
	}
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.ID, invitee.PersonID); err != nil {
		t.Fatal(err)
	}
	member, err := organization.AddTeamUser(ctx, store, owner.ID, team.ID, invitee.PersonID)
	if err != nil {
		t.Fatal(err)
	}
	if member.PersonID != invitee.PersonID {
		t.Fatalf("wrong team membership: %+v", member)
	}
	if _, err := organization.AddTeamUser(ctx, store, owner.ID, team.ID, invitee.PersonID); !errors.Is(err, organization.ErrTeamMemberExists) {
		t.Fatalf("duplicate team user accepted: %v", err)
	}
	teams, err := organization.ListTeams(ctx, store, invitee.ID, org.ID, db.NewListOptions(1, 20, 20, 100))
	if err != nil || len(teams) != 1 {
		t.Fatalf("member team listing: %d / %v", len(teams), err)
	}
	teams, err = organization.ListTeams(ctx, store, outsider.ID, org.ID, db.NewListOptions(1, 20, 20, 100))
	if !errors.Is(err, organization.ErrOrganizationNotFound) || len(teams) != 0 {
		t.Fatalf("outsider must receive not found for private teams: %d / %v", len(teams), err)
	}

	another := makeActor(t, ctx, store)
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.ID, another.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: another.ID, Email: another.Email,
	}); err != nil {
		t.Fatal(err)
	}
	proof, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, another.Email, time.Hour, "01234567890123456789012345678901")
	if err != nil {
		t.Fatal(err)
	}
	if len(proof.Token) != 64 {
		t.Fatalf("unexpected proof size: %d", len(proof.Token))
	}
	accepted, err := organization.AcceptTeamInvite(ctx, store, another.ID, proof.Token)
	if err != nil || accepted.PersonID != another.PersonID {
		t.Fatalf("valid invite rejected: %+v / %v", accepted, err)
	}
	if _, err := organization.AcceptTeamInvite(ctx, store, another.ID, proof.Token); !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("invite replay accepted: %v", err)
	}
	_, err = organization.AcceptTeamInvite(context.Background(), store, outsider.ID, proof.Token)
	if !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("outsider consumed invitation: %v", err)
	}
}

func TestConcurrentTeamInviteConsumption(t *testing.T) {
	ctx, store := integrationStore(t)
	owner := makeActor(t, ctx, store)
	invitee := makeActor(t, ctx, store)
	org, err := organization.CreateOrganization(ctx, store, owner.ID, "Team race "+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.ID, invitee.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{ID: invitee.ID, Email: invitee.Email}); err != nil {
		t.Fatal(err)
	}
	team, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.ID, ActorAccountID: owner.ID, Name: "QA",
	})
	if err != nil {
		t.Fatal(err)
	}
	proof, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, invitee.Email, time.Hour, "01234567890123456789012345678901")
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := organization.AcceptTeamInvite(ctx, store, invitee.ID, proof.Token)
			results <- err
		}()
	}
	successes, invalids := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, organization.ErrTeamInviteInvalid):
			invalids++
		default:
			t.Fatalf("unexpected concurrent invite result: %v", err)
		}
	}
	if successes != 1 || invalids != 1 {
		t.Fatalf("concurrent proof accepted incorrectly: successes=%d invalids=%d", successes, invalids)
	}
}
