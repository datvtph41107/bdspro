package organization

import (
	"fmt"
	"testing"

	org_model "github.com/datvtph41107/bdspro/models/organization"
)

// The PostgreSQL FK added in G1.2c.9a protects owner Account existence, not
// owner count. A privileged raw SQL writer can still demote the last owner.
func TestDoctorDetectsPrivilegedLastOwnerDemotion(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOwnerRoleCase(t, ctx, pool, store, true)

	healthy, err := org_model.CheckOwnerCoverage(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(healthy.OrganizationIDs) != 0 || healthy.HasMore {
		t.Fatalf("unexpected issues in canonical setup: %+v", healthy)
	}

	_, err = pool.Exec(ctx, `UPDATE organization_members
		SET role='member', owner_person_id=NULL
		WHERE organization_id=$1 AND person_id=$2 AND role='owner' AND ended_at IS NULL`,
		d.orgID, d.actorPersonID)
	if err != nil {
		t.Fatalf("corruption fixture was unexpectedly rejected by DB: %v", err)
	}
	requireOwnerCount(t, ctx, pool, d.orgID, 0)

	report, err := org_model.CheckOwnerCoverage(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if report.HasMore || len(report.OrganizationIDs) != 1 || report.OrganizationIDs[0] != d.orgID {
		t.Fatalf("missing ownerless organization after privileged raw demotion: %+v", report)
	}

	// Recovery fixture is direct SQL ONLY in a test; production doctor must
	// not automatically appoint an owner without verified authorization.
	_, err = pool.Exec(ctx, `UPDATE organization_members
		SET role='owner', owner_person_id=person_id
		WHERE organization_id=$1 AND person_id=$2 AND role='member' AND ended_at IS NULL`,
		d.orgID, d.actorPersonID)
	if err != nil {
		t.Fatalf("restore test fixture: %v", err)
	}
	after, err := org_model.CheckOwnerCoverage(ctx, store)
	if err != nil || len(after.OrganizationIDs) != 0 {
		t.Fatalf("healthy organization still flagged: report=%+v err=%v", after, err)
	}
}

func TestDoctorOwnerlessResultIsBoundedAndOrdered(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	firstIDs := make([]int64, 0, 21)
	for i := 0; i < 22; i++ {
		var partyID int64
		if err := pool.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&partyID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO organizations (party_id, name) VALUES ($1,$2)",
			partyID, fmt.Sprintf("Doctor fixture %d", i)); err != nil {
			t.Fatal(err)
		}
		if len(firstIDs) < 21 {
			firstIDs = append(firstIDs, partyID)
		}
	}
	report, err := org_model.CheckOwnerCoverage(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.OrganizationIDs) != 20 || !report.HasMore {
		t.Fatalf("unbounded or nontruncated ownerless diagnostic: %+v", report)
	}
	for i, id := range report.OrganizationIDs {
		if id != firstIDs[i] {
			t.Fatalf("diagnostic not ordered: result[%d]=%d expected=%d", i, id, firstIDs[i])
		}
	}
}

