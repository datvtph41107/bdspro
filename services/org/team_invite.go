package org

import (
	"context"
	"errors"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateTeamInvitation(ctx context.Context, actor authmodel.Actor, orgID, teamID int64, recipient string, seconds int) (organization.TeamInvitation, error) {
	email, err := authservice.NormalizeEmail(recipient)
	if err != nil {
		return organization.TeamInvitation{}, err
	}
	lifetime, err := organization.TeamInviteDuration(seconds)
	if err != nil {
		return organization.TeamInvitation{}, err
	}
	raw, digest, err := token.NewOpaqueProof()
	if err != nil {
		return organization.TeamInvitation{}, err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return organization.TeamInvitation{}, err
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (organization.TeamInvitation, error) {
		q, err := lockTeamOwner(txCtx, actor, orgID, teamID)
		if err != nil {
			return organization.TeamInvitation{}, err
		}
		if !setting.Mailer.Enabled() {
			return organization.TeamInvitation{}, mailer.ErrUnavailable
		}
		if err := q.RevokeExpiredTeamInvites(txCtx, db.RevokeExpiredTeamInvitesParams{TeamID: teamID, InvitedEmail: email}); err != nil {
			return organization.TeamInvitation{}, err
		}
		row, err := q.CreateTeamInviteForOwner(txCtx, db.CreateTeamInviteForOwnerParams{
			ID: id, TeamID: teamID, ActorAccountID: actor.AccountID, InvitedEmail: email,
			TokenHash: digest, ExpiresAt: time.Now().UTC().Add(lifetime),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamInvitation{}, organization.ErrTeamMemberNotFound
		}
		if err != nil {
			return organization.TeamInvitation{}, err
		}
		if err := teamAudit(txCtx, q, actor.AccountID, teamID, "team.invite.created", "invite", id.String()); err != nil {
			return organization.TeamInvitation{}, err
		}
		return organization.TeamInvitation{ID: id.String(), TeamID: decimal(teamID), ExpiresAt: row.ExpiresAt}, nil
	})
	if err != nil {
		return organization.TeamInvitation{}, mapMembershipError(err)
	}

	q, err := db.GetQueries(ctx)
	if err == nil {
		_, err = q.GetTeamInviteForDelivery(ctx, id)
	}
	if err == nil {
		err = mailer.SendTeamInvitation(ctx, email, raw, result.ExpiresAt)
	}
	if err != nil {

		cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		cancelErr := db.WithTx(cancelCtx, func(txCtx context.Context) error {
			q, err := db.GetQueries(txCtx)
			if err != nil {
				return err
			}
			n, err := q.CancelUndeliveredTeamInvite(txCtx, id)
			if err != nil || n == 0 {
				return err
			}
			return teamAudit(txCtx, q, actor.AccountID, teamID, "team.invite.revoked", "invite", id.String())
		})
		if cancelErr != nil {
			return organization.TeamInvitation{}, errors.Join(mailer.ErrUnavailable, cancelErr)
		}
		return organization.TeamInvitation{}, mailer.ErrUnavailable
	}
	return result, nil
}

func RevokeTeamInvitation(ctx context.Context, actor authmodel.Actor, orgID, teamID int64, id uuid.UUID) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := lockTeamOwner(txCtx, actor, orgID, teamID)
		if err != nil {
			return err
		}
		n, err := q.RevokeTeamInviteForOwner(txCtx, db.RevokeTeamInviteForOwnerParams{TeamID: teamID, ActorAccountID: actor.AccountID, InviteID: id})
		if err != nil {
			return err
		}
		if n != 1 {
			return organization.ErrTeamInviteNotFound
		}
		return teamAudit(txCtx, q, actor.AccountID, teamID, "team.invite.revoked", "invite", id.String())
	})
}

func AcceptTeamInvitation(ctx context.Context, actor authmodel.Actor, raw string) (organization.TeamMembership, error) {
	digest, err := token.OpaqueProofDigest(raw)
	if err != nil {
		return organization.TeamMembership{}, organization.ErrTeamInviteInvalid
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (organization.TeamMembership, error) {
		q, err := authservice.LockActorForMutation(txCtx, actor)
		if err != nil {
			return organization.TeamMembership{}, err
		}
		target, err := q.FindTeamInviteOrganization(txCtx, db.FindTeamInviteOrganizationParams{TokenHash: digest, AccountID: actor.AccountID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamMembership{}, organization.ErrTeamInviteInvalid
		}
		if err != nil {
			return organization.TeamMembership{}, err
		}
		if _, err := q.LockOrganizationRoleChanges(txCtx, target.OrganizationID); errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamMembership{}, organization.ErrTeamInviteInvalid
		} else if err != nil {
			return organization.TeamMembership{}, err
		}

		teamID, err := q.ConsumeTeamInvite(txCtx, db.ConsumeTeamInviteParams{TokenHash: digest, AccountID: actor.AccountID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamMembership{}, organization.ErrTeamInviteInvalid
		}
		if err != nil {
			return organization.TeamMembership{}, err
		}
		row, err := q.InsertInvitedTeamUser(txCtx, db.InsertInvitedTeamUserParams{TeamID: teamID, AccountID: actor.AccountID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamMembership{}, organization.ErrTeamInviteInvalid
		}
		if err != nil {
			return organization.TeamMembership{}, err
		}
		if err := teamAudit(txCtx, q, actor.AccountID, teamID, "team.invite.accepted", "invite", target.ID.String()); err != nil {
			return organization.TeamMembership{}, err
		}
		return toTeamMembership(row), nil
	})
	return result, mapMembershipError(err)
}
