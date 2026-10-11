package source

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// AccountResolver must return the account ID from a VERIFIED server-side
// session. There is deliberately no Header-based or anonymous fallback.
// A nil resolver rejects all requests.
type AccountResolver func(*http.Request) (int64, bool)

type captureBody struct {
	Channel      string     `json:"channel"`
	Label        string     `json:"label"`
	AreaM2       string     `json:"area_m2"`
	AreaBasis    string     `json:"area_basis"`
	ReceivedAt   string     `json:"received_at"`
	Summary      string     `json:"summary"`
	InitialPrice *priceBody `json:"initial_price"`
}
type priceBody struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	StatementAt string `json:"statement_at"`
	PriceBasis  string `json:"price_basis"`
	SourceBasis string `json:"source_basis"`
	Note        string `json:"note"`
}

func (p priceBody) toInput() PriceInput {
	return PriceInput{
		Amount: p.Amount, Currency: p.Currency, StatementAt: p.StatementAt, PriceBasis: p.PriceBasis,
		SourceBasis: p.SourceBasis, Note: p.Note,
	}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalid
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		http.Error(w, "invalid input", http.StatusBadRequest)
	case errors.Is(err, ErrKeyConflict):
		http.Error(w, "idempotency key conflict", http.StatusConflict)
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotFound):
		http.Error(w, "resource not found", http.StatusNotFound)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
func positiveID(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, ErrInvalid
	}
	return n, nil
}

// NewHandler is intentionally NOT mounted in main.go on the remote branch:
// existing remote main still uses Listing and has no integrated auth middleware.
// The host application should mount this handler under /organizations/ AFTER
// wiring an AccountResolver from the trusted Session middleware.
func NewHandler(service *Service, resolve AccountResolver) http.Handler {
	mux := http.NewServeMux()
	account := func(w http.ResponseWriter, r *http.Request) (int64, bool) {
		if resolve == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return 0, false
		}
		id, ok := resolve(r)
		if !ok || id <= 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return 0, false
		}
		return id, true
	}
	mux.HandleFunc("POST /organizations/{organizationID}/sources", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := account(w, r)
		if !ok {
			return
		}
		org, err := positiveID(r.PathValue("organizationID"))
		if err != nil {
			writeFailure(w, err)
			return
		}
		var b captureBody
		if err := decodeJSON(w, r, &b); err != nil {
			writeFailure(w, ErrInvalid)
			return
		}
		var price *PriceInput
		if b.InitialPrice != nil {
			p := b.InitialPrice.toInput()
			price = &p
		}
		result, err := service.Capture(r.Context(), actor, org, r.Header.Get("Idempotency-Key"),
			CaptureInput{Channel: b.Channel, Label: b.Label, AreaM2: b.AreaM2, AreaBasis: b.AreaBasis,
				Summary: b.Summary, ReceivedAt: b.ReceivedAt, InitialPrice: price})
		if err != nil {
			writeFailure(w, err)
			return
		}
		status := http.StatusCreated
		if result.Replayed {
			status = http.StatusOK
		}
		writeJSON(w, status, map[string]any{"id": result.ID, "replayed": result.Replayed})
	})
	mux.HandleFunc("GET /organizations/{organizationID}/sources/{sourceID}", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := account(w, r)
		if !ok {
			return
		}
		org, e1 := positiveID(r.PathValue("organizationID"))
		src, e2 := positiveID(r.PathValue("sourceID"))
		if e1 != nil || e2 != nil {
			writeFailure(w, ErrInvalid)
			return
		}
		value, err := service.Read(r.Context(), actor, org, src)
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id": value.ID, "organization_id": value.OrganizationID,
			"recorded_by_account_id": value.RecordedByAccountID, "channel": value.Channel,
			"label": value.Label, "area_m2": value.AreaM2, "area_basis": value.AreaBasis,
			"summary": value.Summary, "version": value.Version, "recorded_at": value.RecordedAt, "received_at": value.ReceivedAt,
		})
	})
	mux.HandleFunc("POST /organizations/{organizationID}/sources/{sourceID}/price-reports", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := account(w, r)
		if !ok {
			return
		}
		org, e1 := positiveID(r.PathValue("organizationID"))
		src, e2 := positiveID(r.PathValue("sourceID"))
		if e1 != nil || e2 != nil {
			writeFailure(w, ErrInvalid)
			return
		}
		var b priceBody
		if err := decodeJSON(w, r, &b); err != nil {
			writeFailure(w, ErrInvalid)
			return
		}
		result, err := service.ReportPrice(r.Context(), actor, org, src, r.Header.Get("Idempotency-Key"), b.toInput())
		if err != nil {
			writeFailure(w, err)
			return
		}
		status := http.StatusCreated
		if result.Replayed {
			status = http.StatusOK
		}
		writeJSON(w, status, map[string]any{"id": result.ID, "replayed": result.Replayed})
	})
	mux.HandleFunc("GET /organizations/{organizationID}/sources/{sourceID}/price-reports", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := account(w, r)
		if !ok {
			return
		}
		org, e1 := positiveID(r.PathValue("organizationID"))
		src, e2 := positiveID(r.PathValue("sourceID"))
		if e1 != nil || e2 != nil {
			writeFailure(w, ErrInvalid)
			return
		}
		history, err := service.PriceHistory(r.Context(), actor, org, src)
		if err != nil {
			writeFailure(w, err)
			return
		}
		out := make([]map[string]any, 0, len(history))
		for _, p := range history {
			out = append(out, map[string]any{
				"id": p.ID, "intake_id": p.IntakeID, "recorded_by_account_id": p.RecordedByAccountID,
				"amount": p.Amount, "currency": p.Currency, "price_basis": p.PriceBasis,
				"source_basis": p.SourceBasis, "note": p.Note, "recorded_at": p.RecordedAt, "statement_at": p.StatementAt,
			})
		}
		writeJSON(w, http.StatusOK, out)
	})
	return mux
}
