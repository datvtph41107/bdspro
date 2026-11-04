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
		OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "Engineering", Description: "Backend",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "engineering",
	}); !errors.Is(err, organization.ErrTeamAlreadyExists) {
		t.Fatalf("case-insensitive duplicate team accepted: %v", err)
	}
	if _, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.PartyID, ActorAccountID: outsider.ID, Name: "Unauthorized",
	}); !errors.Is(err, organization.ErrTeamOwnerRequired) {
		t.Fatalf("non-owner created a team: %v", err)
	}
	if _, err := organization.GetTeam(ctx, store, outsider.ID, team.ID); !errors.Is(err, organization.ErrTeamNotFound) {
		t.Fatalf("outsider read private team: %v", err)
	}
	if _, err := organization.RenameTeam(ctx, store, outsider.ID, team.ID, "Unauthorized rename"); !errors.Is(err, organization.ErrTeamOwnerRequired) {
		t.Fatalf("non-owner renamed a team: %v", err)
	}
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.PartyID, invitee.PersonID); err != nil {
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
	teams, err := organization.ListTeams(ctx, store, invitee.ID, org.PartyID, db.NewListOptions(1, 20, 20, 100))
	if err != nil || len(teams) != 1 {
		t.Fatalf("member team listing: %d / %v", len(teams), err)
	}
	teams, err = organization.ListTeams(ctx, store, outsider.ID, org.PartyID, db.NewListOptions(1, 20, 20, 100))
	if err != nil || len(teams) != 0 {
		t.Fatalf("outsider learned private teams: %d / %v", len(teams), err)
	}

	another := makeActor(t, ctx, store)
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.PartyID, another.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: another.ID, Email: another.Email,
	}); err != nil {
		t.Fatal(err)
	}
	proof, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, another.Email, time.Hour)
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
	if _, err := organization.AddOrgUser(ctx, store, owner.ID, org.PartyID, invitee.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{ID: invitee.ID, Email: invitee.Email}); err != nil {
		t.Fatal(err)
	}
	team, err := organization.CreateTeam(ctx, store, organization.CreateTeamOptions{
		OrganizationID: org.PartyID, ActorAccountID: owner.ID, Name: "QA",
	})
	if err != nil {
		t.Fatal(err)
	}
	proof, err := organization.CreateTeamInvite(ctx, store, owner.ID, team.ID, invitee.Email, time.Hour)
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
