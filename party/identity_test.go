package party

import (
	"context"
	"errors"
	"testing"
)

func TestCreateOrganizationRejectsBlankNameBeforeDatabase(t *testing.T) {
	for _, name := range []string{"", " ", "\t\n"} {
		_, err := CreateOrganization(context.Background(), nil, name)
		if !errors.Is(err, ErrOrganizationNameRequired) {
			t.Errorf("name %q: got %v, want ErrOrganizationNameRequired", name, err)
		}
	}
}
