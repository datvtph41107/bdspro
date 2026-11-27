package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/datvtph41107/bdspro/services/phonedelivery"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func StartContactLink(ctx context.Context, phoneDelivery *phonedelivery.Service, actor authmodel.Actor, kind, destination, proof, channel string) (RegistrationChallenge, error) {
	var code string
	var deliveryChannel phonedelivery.Channel
	var err error
	switch kind {
	case "email":
		if !setting.Mailer.Enabled() {
			return RegistrationChallenge{}, mailer.ErrUnavailable
		}
		destination, err = NormalizeEmail(destination)
		if err != nil {
			return RegistrationChallenge{}, err
		}
		code, err = token.NewRegistrationCode()
	case "phone":
		deliveryChannel, err = phoneDelivery.ResolveChannel(channel)
		if err != nil {
			return RegistrationChallenge{}, err
		}
		destination, err = token.NormalizePhone(destination, "")
		if err != nil {
			return RegistrationChallenge{}, ErrPhoneInvalid
		}
		code, err = token.NewPhoneCode()
	default:
		return RegistrationChallenge{}, ErrReauthAction
	}
	if err != nil {
		return RegistrationChallenge{}, err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if _, err := q.DeleteExpiredCredentialLinks(ctx, 100); err != nil {
		return RegistrationChallenge{}, err
	}
	err = db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return err
		}
		if err := ConsumeRecentAuthProof(txCtx, q, actor, proof, kind+".link"); err != nil {
			return err
		}

		n, err := q.StartCredentialLink(txCtx, db.StartCredentialLinkParams{
			ID: id, AccountID: actor.AccountID, SessionID: actor.SessionID, Kind: kind,
			Destination: destination,
			CodeMac:     token.ContactLinkCodeMAC(setting.Security.TokenSymmetricKey, id, actor.AccountID, actor.SessionID, kind, destination, code),
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrRegistrationWait
		}
		return nil
	})
	if err != nil {
		return RegistrationChallenge{}, err
	}

	if kind == "email" {
		err = mailer.SendRegistrationCode(ctx, destination, code)
	} else {
		_, err = phoneDelivery.SendCode(ctx, deliveryChannel, destination, code)
	}
	if err != nil {
		return RegistrationChallenge{}, err
	}
	result := challengeResult(id)
	result.DeliveryChannel = string(deliveryChannel)
	return result, nil
}

func ContactLinkDestination(ctx context.Context, actor authmodel.Actor, kind string, id uuid.UUID) (string, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return "", err
	}
	destination, err := q.ReadCredentialLinkDestination(ctx, db.ReadCredentialLinkDestinationParams{
		ID: id, AccountID: actor.AccountID, SessionID: actor.SessionID, Kind: kind,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRegistrationCode
	}
	return destination, err
}

func VerifyContactLink(ctx context.Context, actor authmodel.Actor, kind string, id uuid.UUID, code, ip, agent string) (LoginResult, error) {
	if kind != "email" && kind != "phone" {
		return LoginResult{}, ErrReauthAction
	}
	var changed bool
	var oldEmail string
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return LoginResult{}, err
		}
		row, err := q.LockCredentialLink(txCtx, db.LockCredentialLinkParams{
			ID: id, AccountID: actor.AccountID, SessionID: actor.SessionID, Kind: kind,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, nil
		}
		if err != nil {
			return LoginResult{}, err
		}
		if row.FailedAttempts >= 5 || !row.ExpiresAt.After(time.Now()) {
			return LoginResult{}, nil
		}
		expected := token.ContactLinkCodeMAC(setting.Security.TokenSymmetricKey, id, actor.AccountID, actor.SessionID, kind, row.Destination, code)
		validCode := kind == "email" && token.ValidRegistrationCode(code) || kind == "phone" && token.ValidPhoneCode(code)
		if !validCode || !hmac.Equal(expected, row.CodeMac) {
			return LoginResult{}, q.FailCredentialLink(txCtx, id)
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return LoginResult{}, err
		}
		if u.Email != nil && u.Credentials.EmailVerified {
			oldEmail = *u.Email
		}
		switch kind {
		case "email":
			owner, err := q.ReadAccountByEmail(txCtx, row.Destination)
			if err == nil && owner.ID != actor.AccountID {
				return LoginResult{}, ErrCredentialTaken
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return LoginResult{}, err
			}
			n, err := q.SetVerifiedAccountEmail(txCtx, db.SetVerifiedAccountEmailParams{AccountID: actor.AccountID, Email: row.Destination})
			if err != nil {
				return LoginResult{}, err
			}
			if n != 1 {
				return LoginResult{}, util.ErrUnauthenticated
			}
		case "phone":
			owner, err := q.GetPhoneAccountIDQuery(txCtx, row.Destination)
			if err == nil && owner != actor.AccountID {
				return LoginResult{}, ErrCredentialTaken
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return LoginResult{}, err
			}
			if err := q.SetVerifiedAccountPhone(txCtx, db.SetVerifiedAccountPhoneParams{AccountID: actor.AccountID, PhoneE164: row.Destination}); err != nil {
				return LoginResult{}, err
			}
		}
		if err := invalidateCredentialAuthority(txCtx, q, actor.AccountID); err != nil {
			return LoginResult{}, err
		}
		if err := credentialAudit(txCtx, q, actor.AccountID, "auth.credential.linked", kind); err != nil {
			return LoginResult{}, err
		}
		changed = true
		return issueSession(txCtx, q, actor.AccountID, ip, agent, false)
	})
	if err != nil {
		return LoginResult{}, identityConflict(err)
	}
	if !changed {
		return LoginResult{}, ErrRegistrationCode
	}
	notifyCredentialChange(ctx, oldEmail)
	return result, nil
}
