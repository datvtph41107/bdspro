package api

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/source"
	"github.com/gin-gonic/gin"
)

type captureSourceRequest struct {
	Channel      string              `json:"channel"`
	Label        string              `json:"label"`
	AreaM2       string              `json:"area_m2"`
	AreaBasis    string              `json:"area_basis"`
	ReceivedAt   string              `json:"received_at"`
	Summary      string              `json:"summary"`
	InitialPrice *sourcePriceRequest `json:"initial_price"`
}

type sourcePriceRequest struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	StatementAt string `json:"statement_at"`
	PriceBasis  string `json:"price_basis"`
	SourceBasis string `json:"source_basis"`
	Note        string `json:"note"`
}

func (p sourcePriceRequest) toInput() source.PriceInput {
	return source.PriceInput{
		Amount: p.Amount, Currency: p.Currency, StatementAt: p.StatementAt,
		PriceBasis: p.PriceBasis, SourceBasis: p.SourceBasis, Note: p.Note,
	}
}

func sourceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, source.ErrInvalid):
		ctx.String(http.StatusBadRequest, "invalid input")
	case errors.Is(err, source.ErrKeyConflict):
		ctx.String(http.StatusConflict, "idempotency key conflict")
	case errors.Is(err, source.ErrForbidden), errors.Is(err, source.ErrNotFound):
		ctx.String(http.StatusNotFound, "resource not found")
	default:
		ctx.String(http.StatusInternalServerError, "internal server error")
	}
}

func (server *Server) sourceIDs(ctx *gin.Context) (int64, int64, bool) {
	orgID, e1 := auth.PositiveID(ctx.Param("id"))
	sourceID, e2 := auth.PositiveID(ctx.Param("sourceID"))
	if e1 != nil || e2 != nil {
		ctx.String(http.StatusBadRequest, "invalid input")
		return 0, 0, false
	}
	return orgID, sourceID, true
}

func (server *Server) captureSource(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok { return }
	orgID, err := auth.PositiveID(ctx.Param("id"))
	if err != nil { sourceError(ctx, source.ErrInvalid); return }
	var req captureSourceRequest
	if !bindJSON(ctx, &req, 1<<20) {
		sourceError(ctx, source.ErrInvalid)
		return
	}
	var price *source.PriceInput
	if req.InitialPrice != nil {
		p := req.InitialPrice.toInput()
		price = &p
	}
	result, err := server.sources.Capture(ctx.Request.Context(), actor, orgID, ctx.GetHeader("Idempotency-Key"),
		source.CaptureInput{
			Channel: req.Channel, Label: req.Label, AreaM2: req.AreaM2,
			AreaBasis: req.AreaBasis, Summary: req.Summary,
			ReceivedAt: req.ReceivedAt, InitialPrice: price,
		})
	if err != nil { sourceError(ctx, err); return }
	status := http.StatusCreated
	if result.Replayed { status = http.StatusOK }
	replyJSON(ctx, status, gin.H{"id": result.ID, "replayed": result.Replayed})
}

func (server *Server) readSource(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok { return }
	orgID, sourceID, ok := server.sourceIDs(ctx)
	if !ok { return }
	value, err := server.sources.Read(ctx.Request.Context(), actor, orgID, sourceID)
	if err != nil { sourceError(ctx, err); return }
	replyJSON(ctx, http.StatusOK, gin.H{
		"id": value.ID, "organization_id": value.OrganizationID,
		"recorded_by_account_id": value.RecordedByAccountID,
		"channel": value.Channel, "label": value.Label,
		"area_m2": value.AreaM2, "area_basis": value.AreaBasis,
		"summary": value.Summary, "version": value.Version,
		"recorded_at": value.RecordedAt, "received_at": value.ReceivedAt,
	})
}

func (server *Server) reportSourcePrice(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok { return }
	orgID, sourceID, ok := server.sourceIDs(ctx)
	if !ok { return }
	var req sourcePriceRequest
	if !bindJSON(ctx, &req, 1<<20) {
		sourceError(ctx, source.ErrInvalid)
		return
	}
	result, err := server.sources.ReportPrice(ctx.Request.Context(), actor, orgID, sourceID,
		ctx.GetHeader("Idempotency-Key"), req.toInput())
	if err != nil { sourceError(ctx, err); return }
	status := http.StatusCreated
	if result.Replayed { status = http.StatusOK }
	replyJSON(ctx, status, gin.H{"id": result.ID, "replayed": result.Replayed})
}

func (server *Server) listSourcePrices(ctx *gin.Context) {
	actor, ok := server.currentAccount(ctx)
	if !ok { return }
	orgID, sourceID, ok := server.sourceIDs(ctx)
	if !ok { return }
	history, err := server.sources.PriceHistory(ctx.Request.Context(), actor, orgID, sourceID)
	if err != nil { sourceError(ctx, err); return }
	out := make([]gin.H, 0, len(history))
	for _, p := range history {
		out = append(out, gin.H{
			"id": p.ID, "intake_id": p.IntakeID,
			"recorded_by_account_id": p.RecordedByAccountID,
			"amount": p.Amount, "currency": p.Currency,
			"price_basis": p.PriceBasis, "source_basis": p.SourceBasis,
			"note": p.Note, "recorded_at": p.RecordedAt,
			"statement_at": p.StatementAt,
		})
	}
	replyJSON(ctx, http.StatusOK, out)
}
