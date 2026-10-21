package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/datvtph41107/bdspro/util"
)

// The input boundary canonicalizes values BEFORE idempotency hashing.
// These checks do not replace PostgreSQL constraints or SQL authorization.
func preparePrice(p PriceInput) (PriceInput, error) {
	var err error
	p.Amount, err = util.DecimalString(p.Amount, 15, 4)
	if err != nil {
		return PriceInput{}, fmt.Errorf("%w: price amount", ErrInvalid)
	}
	p.Currency, err = util.CurrencyCode(p.Currency)
	if err != nil {
		return PriceInput{}, fmt.Errorf("%w: currency", ErrInvalid)
	}
	p.StatementAt, err = util.TimestampUTC(p.StatementAt)
	if err != nil {
		return PriceInput{}, fmt.Errorf("%w: statement_at", ErrInvalid)
	}
	p.PriceBasis = strings.TrimSpace(p.PriceBasis)
	p.SourceBasis = strings.TrimSpace(p.SourceBasis)
	p.Note = strings.TrimSpace(p.Note)
	if len(p.Note) > 4000 {
		return PriceInput{}, fmt.Errorf("%w: price note too long", ErrInvalid)
	}
	return p, nil
}

func prepareCapture(in CaptureInput) (CaptureInput, error) {
	in.Channel = strings.ToLower(strings.TrimSpace(in.Channel))
	in.Label = strings.TrimSpace(in.Label)
	in.AreaM2 = strings.TrimSpace(in.AreaM2)
	in.AreaBasis = strings.TrimSpace(in.AreaBasis)
	in.Summary = strings.TrimSpace(in.Summary)
	var err error
	in.ReceivedAt, err = util.TimestampUTC(in.ReceivedAt)
	if err != nil {
		return CaptureInput{}, fmt.Errorf("%w: received_at", ErrInvalid)
	}
	if in.Channel == "" || len(in.Channel) > 50 || len(in.Label) > 400 ||
		len(in.Summary) > 4000 || len(in.AreaBasis) > 80 {
		return CaptureInput{}, fmt.Errorf("%w: source metadata", ErrInvalid)
	}
	if in.AreaM2 != "" {
		in.AreaM2, err = util.DecimalString(in.AreaM2, 14, 4)
		if err != nil {
			return CaptureInput{}, fmt.Errorf("%w: area", ErrInvalid)
		}
	}
	if in.InitialPrice != nil {
		p, err := preparePrice(*in.InitialPrice)
		if err != nil {
			return CaptureInput{}, err
		}
		in.InitialPrice = &p
	}
	return in, nil
}

func validateKey(key string) error {
	if len(key) == 0 || len(key) > 100 || strings.TrimSpace(key) != key {
		return fmt.Errorf("%w: Idempotency-Key must be 1..100 bytes", ErrInvalid)
	}
	return nil
}

// hashRequest belongs to Source because its exact JSON digest is a durable
// part of the Source idempotency contract, not a generic application hash.
func hashRequest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
