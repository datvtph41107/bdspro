package auth

import (
	"context"
	"errors"
	"strconv"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

func GetCurrentUser(ctx context.Context, accountID int64) (authmodel.CurrentUser, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	a, err := q.ReadAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return authmodel.CurrentUser{}, util.ErrUnauthenticated
	}
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	identity, err := q.GetAccountIdentity(ctx, accountID)
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	if identity.Status != "active" {
		return authmodel.CurrentUser{}, util.ErrUnauthenticated
	}
	u := authmodel.CurrentUser{
		ID: strconv.FormatInt(a.ID, 10), PersonID: strconv.FormatInt(a.PersonID, 10),
		Status:     identity.Status,
		Onboarding: authmodel.Onboarding{RequiredActions: []string{}},
	}
	if identity.Username == "" {
		u.Onboarding.RequiredActions = append(u.Onboarding.RequiredActions, "choose_username")
	} else {
		u.Username = &identity.Username
	}
	if identity.TermsVersion != CurrentTermsVersion {
		u.Onboarding.RequiredActions = append(u.Onboarding.RequiredActions, "accept_terms")
	}
	u.Onboarding.Completed = len(u.Onboarding.RequiredActions) == 0
	if a.Email != "" {
		u.Email = &a.Email
	}
	u.Credentials.EmailVerified = a.IsEmailVerified
	p, err := GetProfile(ctx, accountID)
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	u.DisplayName = p.DisplayName
	if _, err = q.GetPasswordHash(ctx, accountID); err == nil {
		u.Credentials.PasswordSet = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return authmodel.CurrentUser{}, err
	}
	phone, err := q.GetPhoneForAccount(ctx, accountID)
	if err == nil {
		u.Phone = &phone
		u.Credentials.PhoneVerified = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return authmodel.CurrentUser{}, err
	}
	identities, err := q.ListOAuth2CredentialsForAccount(ctx, accountID)
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	for _, identity := range identities {
		if identity.Issuer == "https://accounts.google.com" || identity.Issuer == "accounts.google.com" {
			u.Credentials.GoogleLinked = true
			break
		}
	}
	return u, nil
}
