package oauth

import (
	"net/http"

	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
)

// Authorization ownership and client ownership remain distinct. Only the
// authenticated account's consent grants are listed or revoked.
func ListAuthorizations(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	grants,err:=s.ListAuthorizations(ctx,accountID)
	if err!=nil {serviceFailure(ctx,err);return}
	out:=make([]map[string]any,0,len(grants))
	for _,grant:=range grants {
		out=append(out,map[string]any{
			"client_id":grant.ClientID.String(),"name":grant.Name,"scope":grant.Scope,
		})
	}
	ctx.Header("Cache-Control","no-store")
	ctx.JSON(http.StatusOK,map[string]any{"authorizations":out})
}

func RevokeAuthorization(ctx *appctx.APIContext) {
	s,ok:=service(ctx);if !ok{return}
	accountID,ok:=actorID(ctx);if !ok{return}
	clientID,err:=uuid.Parse(ctx.PathParam("client_id"))
	if err!=nil {failure(ctx,http.StatusBadRequest,"invalid_request");return}
	if err:=s.RevokeAuthorization(ctx,accountID,clientID);err!=nil {
		serviceFailure(ctx,err);return
	}
	ctx.Header("Cache-Control","no-store")
	ctx.Status(http.StatusNoContent)
}
