package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrForbidden = errors.New("organization operation not permitted")
	ErrInvalid   = errors.New("invalid organization request")
	ErrNotFound  = errors.New("organization request unavailable")
	ErrConflict  = errors.New("organization conflict or duplicate")
)

type Service struct{ Pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{Pool: pool} }
func requireAccount(accountID int64) error {
	if accountID <= 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Service) SubmitRegistration(ctx context.Context, accountID int64, name, jurisdiction, registration string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	name = strings.TrimSpace(name)
	jurisdiction = strings.ToUpper(strings.TrimSpace(jurisdiction))
	registration = strings.TrimSpace(registration)
	if name == "" || jurisdiction == "" || len(name) > 250 || len(jurisdiction) > 50 || len(registration) > 100 {
		return 0, ErrInvalid
	}
	var id int64
	err := s.Pool.QueryRow(ctx, `INSERT INTO organization_registration_requests(requested_by_account_id,declared_name,declared_jurisdiction_code,declared_registration_code)
 SELECT a.id,$2,$3,NULLIF($4,'') FROM accounts a
 WHERE a.id=$1 AND a.closed_at IS NULL AND EXISTS(
   SELECT 1 FROM account_emails ae WHERE ae.account_id=a.id AND ae.verified_at IS NOT NULL)
 RETURNING id`, accountID, name, jurisdiction, registration).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}
func (s *Service) SubmitClaim(ctx context.Context, accountID, organizationID int64, evidence string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	evidence = strings.TrimSpace(evidence)
	if organizationID <= 0 || evidence == "" || len(evidence) > 1000 {
		return 0, ErrInvalid
	}
	var id int64
	err := s.Pool.QueryRow(ctx, `INSERT INTO organization_claim_requests(organization_id,requested_by_account_id,evidence_reference)
 SELECT o.party_id,a.id,$3 FROM organizations o JOIN accounts a ON a.id=$1
 WHERE o.party_id=$2 AND o.archived_at IS NULL AND a.closed_at IS NULL
   AND EXISTS(SELECT 1 FROM account_emails ae WHERE ae.account_id=a.id AND ae.verified_at IS NOT NULL)
 RETURNING id`, accountID, organizationID, evidence).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}

// Only an operator (NOT tenant admin) can provision platform_organization_reviewers.
// Reviewer authority is checked and row-locked in the same transaction as approval.
func requireReviewer(ctx context.Context, tx pgx.Tx, accountID int64) error {
	var id int64
	err := tx.QueryRow(ctx, `SELECT r.account_id FROM platform_organization_reviewers r
 JOIN accounts a ON a.id=r.account_id
 WHERE r.account_id=$1 AND r.revoked_at IS NULL AND a.closed_at IS NULL
 FOR SHARE OF r,a`, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	return err
}
func grantInitialAdmin(ctx context.Context, tx pgx.Tx, orgID, personID int64) error {
	var membershipID, roleID int64
	err := tx.QueryRow(ctx, `INSERT INTO organization_memberships(organization_id,person_id)
 VALUES($1,$2) ON CONFLICT(organization_id,person_id) WHERE ended_at IS NULL
 DO UPDATE SET person_id=EXCLUDED.person_id
 RETURNING id`, orgID, personID).Scan(&membershipID)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, "INSERT INTO organization_roles(organization_id,name) VALUES($1,'Workspace Administrator') RETURNING id", orgID).Scan(&roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization_role_permissions(role_id,permission_code)
 SELECT $1,code FROM permissions WHERE code IN(
 'members.read','members.invite','sources.create','sources.read','sources.price_report')`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization_role_assignments(organization_id,membership_id,role_id)
 VALUES($1,$2,$3)`, orgID, membershipID, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO organization_primary_administrators(organization_id,membership_id) VALUES($1,$2)", orgID, membershipID)
	return err
}

// ApproveRegistration atomically creates Party+Organization, validated registry
// identity, tenant admin membership/role and a resolved review request.
func (s *Service) ApproveRegistration(ctx context.Context, reviewerID, requestID int64, reference string) (int64, error) {
	if reviewerID <= 0 || requestID <= 0 || strings.TrimSpace(reference) == "" {
		return 0, ErrInvalid
	}
	var orgID int64
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := requireReviewer(ctx, tx, reviewerID); err != nil {
			return err
		}
		var requester int64
		var name, jurisdiction string
		var registration *string
		err := tx.QueryRow(ctx, `SELECT requested_by_account_id,declared_name,declared_jurisdiction_code,declared_registration_code
 FROM organization_registration_requests WHERE id=$1 AND decision_code IS NULL FOR UPDATE`, requestID).
			Scan(&requester, &name, &jurisdiction, &registration)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if registration == nil || strings.TrimSpace(*registration) == "" {
			return fmt.Errorf("%w: verified registry identifier required", ErrInvalid)
		}
		var personID int64
		err = tx.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1 AND closed_at IS NULL", requester).Scan(&personID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, "INSERT INTO parties(kind) VALUES('ORGANIZATION') RETURNING id").Scan(&orgID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO organizations(party_id,name) VALUES($1,$2)", orgID, name); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO organization_registry_identifiers(organization_id,jurisdiction_code,registration_code,verified_by_account_id)
 VALUES($1,$2,$3,$4)`, orgID, jurisdiction, strings.TrimSpace(*registration), reviewerID); err != nil {
			return err
		}
		if err = grantInitialAdmin(ctx, tx, orgID, personID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE organization_registration_requests SET
 decision_code='APPROVED',resolved_at=now(),matched_organization_id=$1,
 reviewed_by_account_id=$2,review_reference=$3 WHERE id=$4 AND decision_code IS NULL`,
			orgID, reviewerID, strings.TrimSpace(reference), requestID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23503") {
			return 0, ErrConflict
		}
		return 0, err
	}
	return orgID, nil
}

// A claim can bootstrap the FIRST workspace admin only when no administrator
// exists. Never silently steal/replace a tenant's incumbent admin.
func (s *Service) ApproveInitialClaim(ctx context.Context, reviewerID, requestID int64, reference string) (int64, error) {
	if reviewerID <= 0 || requestID <= 0 || strings.TrimSpace(reference) == "" {
		return 0, ErrInvalid
	}
	var orgID int64
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := requireReviewer(ctx, tx, reviewerID); err != nil {
			return err
		}
		var applicant int64
		err := tx.QueryRow(ctx, `SELECT organization_id,requested_by_account_id
 FROM organization_claim_requests WHERE id=$1 AND decision_code IS NULL FOR UPDATE`, requestID).
			Scan(&orgID, &applicant)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Organization row serializes competing initial claims of the same company.
		var locked int64
		if err := tx.QueryRow(ctx, "SELECT party_id FROM organizations WHERE party_id=$1 AND archived_at IS NULL FOR UPDATE", orgID).Scan(&locked); err != nil {
			return ErrConflict
		}
		var primary int64
		err = tx.QueryRow(ctx, "SELECT membership_id FROM organization_primary_administrators WHERE organization_id=$1", orgID).Scan(&primary)
		if err == nil {
			return ErrConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var personID int64
		if err = tx.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1 AND closed_at IS NULL", applicant).Scan(&personID); err != nil {
			return ErrConflict
		}
		if err = grantInitialAdmin(ctx, tx, orgID, personID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE organization_claim_requests SET decision_code='APPROVED',
 resolved_at=now(),reviewed_by_account_id=$1,review_reference=$2
 WHERE id=$3 AND decision_code IS NULL`, reviewerID, strings.TrimSpace(reference), requestID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return orgID, nil
}

func (s *Service) Invite(ctx context.Context, accountID, organizationID, roleID int64, email string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if organizationID <= 0 || roleID <= 0 || len(email) > 254 || email == "" || !strings.Contains(email, "@") {
		return 0, ErrInvalid
	}
	var id int64
	err := s.Pool.QueryRow(ctx, `INSERT INTO organization_invitations(organization_id,invitee_email,intended_role_id,invited_by_account_id,expires_at)
 SELECT o.party_id,$4,role.id,a.id,now()+interval '7 days'
 FROM organizations o JOIN organization_roles role ON role.organization_id=o.party_id AND role.id=$3
 JOIN accounts a ON a.id=$1
 WHERE o.party_id=$2 AND o.archived_at IS NULL AND a.closed_at IS NULL AND EXISTS(
 SELECT 1 FROM organization_memberships m
 JOIN organization_role_assignments ra ON ra.organization_id=m.organization_id AND ra.membership_id=m.id
 JOIN organization_role_permissions rp ON rp.role_id=ra.role_id
 WHERE m.organization_id=o.party_id AND m.person_id=a.person_id
 AND m.joined_at<=now() AND m.ended_at IS NULL
 AND ra.assigned_at<=now() AND ra.ended_at IS NULL
 AND rp.permission_code='members.invite'
 AND NOT EXISTS(SELECT 1 FROM organization_membership_blocks b WHERE b.membership_id=m.id AND b.ended_at IS NULL AND b.blocked_at<=now()))
 RETURNING id`, accountID, organizationID, roleID, email).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}
func (s *Service) AcceptInvitation(ctx context.Context, accountID, invitationID int64) (int64, error) {
	if accountID <= 0 || invitationID <= 0 {
		return 0, ErrInvalid
	}
	var membership int64
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var orgID, roleID int64
		err := tx.QueryRow(ctx, `UPDATE organization_invitations i SET outcome='ACCEPTED',resolved_at=now()
 WHERE i.id=$1 AND i.outcome IS NULL AND i.expires_at>now()
 AND EXISTS(SELECT 1 FROM accounts a JOIN account_emails ae ON ae.account_id=a.id
 WHERE a.id=$2 AND a.closed_at IS NULL AND ae.verified_at IS NOT NULL
 AND lower(btrim(ae.email))=lower(btrim(i.invitee_email)))
 RETURNING i.organization_id,i.intended_role_id`, invitationID, accountID).Scan(&orgID, &roleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var person int64
		if err := tx.QueryRow(ctx, "SELECT person_id FROM accounts WHERE id=$1", accountID).Scan(&person); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO organization_memberships(organization_id,person_id,originating_invitation_id)
 VALUES($1,$2,$3) RETURNING id`, orgID, person, invitationID).Scan(&membership); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO organization_role_assignments(organization_id,membership_id,role_id)
 VALUES($1,$2,$3)`, orgID, membership, roleID)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrConflict
		}
		return 0, err
	}
	return membership, nil
}

type Member struct {
	MembershipID int64
	PersonID     int64
	JoinedAt     time.Time
}

func (s *Service) ListMembers(ctx context.Context, accountID, organizationID int64) ([]Member, error) {
	if accountID <= 0 || organizationID <= 0 {
		return nil, ErrForbidden
	}
	rows, err := s.Pool.Query(ctx, `SELECT target.id,target.person_id,target.joined_at FROM organization_memberships target
 JOIN organizations o ON o.party_id=target.organization_id
 JOIN accounts actor ON actor.id=$1
 WHERE target.organization_id=$2 AND target.ended_at IS NULL AND target.joined_at<=now()
 AND o.archived_at IS NULL AND actor.closed_at IS NULL AND EXISTS(
 SELECT 1 FROM organization_memberships m
 JOIN organization_role_assignments ra ON ra.organization_id=m.organization_id AND ra.membership_id=m.id
 JOIN organization_role_permissions rp ON rp.role_id=ra.role_id
 WHERE m.organization_id=o.party_id AND m.person_id=actor.person_id
 AND m.ended_at IS NULL AND m.joined_at<=now()
 AND ra.ended_at IS NULL AND ra.assigned_at<=now() AND rp.permission_code='members.read'
 AND NOT EXISTS(SELECT 1 FROM organization_membership_blocks b WHERE b.membership_id=m.id AND b.ended_at IS NULL AND b.blocked_at<=now()))
 ORDER BY target.id`, accountID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.MembershipID, &m.PersonID, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, ErrNotFound
	}
	return members, nil
}
