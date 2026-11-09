package organization

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeOwnerCoverageReader struct {
	ids []int64
	err error
}

func (f fakeOwnerCoverageReader) ListOrganizationsWithoutActiveOwner(context.Context, int64) ([]int64, error) {
	return f.ids, f.err
}

func TestCheckOwnerCoverage(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		report, err := CheckOwnerCoverage(context.Background(), fakeOwnerCoverageReader{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.OrganizationIDs) != 0 || report.HasMore {
			t.Fatalf("healthy owner coverage report: %+v", report)
		}
	})
	t.Run("few affected organizations", func(t *testing.T) {
		report, err := CheckOwnerCoverage(context.Background(), fakeOwnerCoverageReader{ids: []int64{2, 5}}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(report.OrganizationIDs, []int64{2, 5}) || report.HasMore {
			t.Fatalf("unbounded/wrong report: %+v", report)
		}
	})
	t.Run("bounded with explicit truncation", func(t *testing.T) {
		ids := make([]int64, 21)
		for i := range ids {
			ids[i] = int64(i + 1)
		}
		report, err := CheckOwnerCoverage(context.Background(), fakeOwnerCoverageReader{ids: ids}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.OrganizationIDs) != maxReportedOwnerlessOrganizations || !report.HasMore {
			t.Fatalf("cap/truncation lost: %+v", report)
		}
		if report.OrganizationIDs[0] != 1 || report.OrganizationIDs[19] != 20 {
			t.Fatalf("wrong first 20 IDs: %+v", report)
		}
	})
	t.Run("negative cursor rejected", func(t *testing.T) {
		if _, err := CheckOwnerCoverage(context.Background(), fakeOwnerCoverageReader{}, -1); err == nil {
			t.Fatal("negative cursor accepted")
		}
	})
	t.Run("database error must fail closed", func(t *testing.T) {
		want := errors.New("database offline")
		report, err := CheckOwnerCoverage(context.Background(), fakeOwnerCoverageReader{err: want}, 0)
		if !errors.Is(err, want) || len(report.OrganizationIDs) != 0 {
			t.Fatalf("database error represented as healthy: report=%+v err=%v", report, err)
		}
	})
}
