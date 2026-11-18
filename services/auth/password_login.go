package auth

import (
	"context"
	"errors"
	"time"

	token "github.com/datvtph41107/bdspro/modules/auth"
)

// LoginPassword is the only local password login orchestration:
// credential verification precedes any session issuance. It is not a
// public HTTP entrypoint and does not bypass HTTP admission controls.
func LoginPassword(
	ctx context.Context,
	email, password, userAgent, clientIP string,
	maker token.Maker,
	accessTTL, refreshTTL time.Duration,
) (SessionTokens, error) {
	if ctx == nil {
		return SessionTokens{}, errors.New("auth: nil context")
	}
	if err := ctx.Err(); err != nil {
		return SessionTokens{}, err
	}
	accountID, err := VerifyPasswordProof(ctx, email, password)
	if err != nil {
		return SessionTokens{}, err
	}
	return IssueSession(ctx, accountID, userAgent, clientIP, maker, accessTTL, refreshTTL)
}
