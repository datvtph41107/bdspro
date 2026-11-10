package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/google/uuid"
)

// ConsentPreview is the domain result of a non-consuming OAuth consent review.
// HTML and JSON routers may render it differently without duplicating proof
// issuance, version checks or registered-client validation.
type ConsentPreview struct {
	Client  *auth_model.OAuthClient
	Request AuthorizationRequest
	Proof   string
}

var ErrConsentProofUnavailable = errors.New("OAuth consent proof service unavailable")

// consentFields binds every value displayed to the signed, account-scoped
// consent decision. Neither HTTP frontend constructs a proof independently.
func consentFields(accountID int64, req AuthorizationRequest) ConsentProofFields {
	return ConsentProofFields{
		AccountID: accountID, ClientID: req.ClientID.String(),
		RedirectURI: req.RedirectURI, Scope: req.Scope,
		ResponseType: req.ResponseType, CodeChallenge: req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		State: req.State, ConsentVersion: req.ConsentVersion,
	}
}

func (s *Service) PreviewConsent(ctx context.Context, accountID int64, req AuthorizationRequest, proofs *ConsentProofs, now time.Time) (ConsentPreview, error) {
	client, version, err := s.ReviewConsent(ctx, accountID, req)
	if err != nil {
		return ConsentPreview{}, err
	}
	if proofs == nil {
		return ConsentPreview{}, ErrConsentProofUnavailable
	}
	req.ConsentVersion = version
	proof, err := proofs.Issue(consentFields(accountID, req), now)
	if err != nil {
		return ConsentPreview{}, err
	}
	return ConsentPreview{Client: client, Request: req, Proof: proof}, nil
}

// ParseConsentDecision parses protocol parameters only. HTTP callers own body
// limits and media type policy; this service owns the OAuth field semantics.
func ParseConsentDecision(form url.Values) (AuthorizationRequest, bool, string, error) {
	req, err := ParseAuthorizationRequest(form)
	if err != nil {
		return AuthorizationRequest{}, false, "", err
	}
	approve := form.Get("approve") == "true"
	if approve {
		version, err := strconv.ParseInt(form.Get("consent_version"), 10, 64)
		if err != nil || version < 0 {
			return AuthorizationRequest{}, false, "", ErrInvalidRequest
		}
		req.ConsentVersion = version
	}
	return req, approve, form.Get("consent_proof"), nil
}

// DecideConsent validates the signed preview and registered redirect before
// issuing any authorization code. Browser CSRF/origin checks remain HTTP
// responsibilities and must precede this domain operation.
func (s *Service) DecideConsent(ctx context.Context, accountID int64, sessionID uuid.UUID, req AuthorizationRequest, approve bool, proof string, proofs *ConsentProofs, now time.Time) (string, error) {
	if approve {
		if proofs == nil || proofs.Verify(proof, consentFields(accountID, req), now) != nil {
			return "", ErrInvalidRequest
		}
	}
	if _, err := s.ReviewAuthorization(ctx, req); err != nil {
		return "", err
	}
	if !approve {
		target, err := url.Parse(req.RedirectURI)
		if err != nil {
			return "", fmt.Errorf("parse registered OAuth redirect: %w", err)
		}
		query := target.Query()
		query.Set("error", "access_denied")
		query.Set("state", req.State)
		target.RawQuery = query.Encode()
		return target.String(), nil
	}
	return s.Authorize(ctx, accountID, sessionID, req)
}
