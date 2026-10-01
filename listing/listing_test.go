package listing

import (
	"slices"
	"testing"
)

func TestPublicationReadiness(t *testing.T) {
	description := "Căn hộ 2 phòng ngủ"
	blankDescription := "      "

	tests := []struct {
		name        string
		listing     Listing
		wantReady   bool
		wantMissing []string
	}{
		{
			name: "draft without description is not ready",
			listing: Listing{
				Status:      "DRAFT",
				Description: nil,
			},
			wantReady:   false,
			wantMissing: []string{"description"},
		},
		{
			name: "draft with blank description is not ready",
			listing: Listing{
				Status:      "DRAFT",
				Description: &blankDescription,
			},
			wantReady:   false,
			wantMissing: []string{"description"},
		},
		{
			name: "complete draft is ready",
			listing: Listing{
				Status:      "DRAFT",
				Description: &description,
			},
			wantReady:   true,
			wantMissing: []string{},
		},
		{
			name: "published listing is not ready",
			listing: Listing{
				Status:      "PUBLISHED",
				Description: &description,
			},
			wantReady:   false,
			wantMissing: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := publicationReadiness(tt.listing)

			if got.Ready != tt.wantReady {
				t.Fatalf(
					"Ready = %v, want %v",
					got.Ready,
					tt.wantReady,
				)
			}

			if !slices.Equal(got.Missing, tt.wantMissing) {
				t.Fatalf(
					"Missing = %v, want %v",
					got.Missing,
					tt.wantMissing,
				)
			}
		})
	}
}
