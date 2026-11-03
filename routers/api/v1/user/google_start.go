package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/datvtph41107/bdspro/modules/structs"
	middleware "github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
)

type googleLinkStartRequest = structs.GoogleLinkRequest

func (g *GoogleHandler) StartLink(ctx *gin.Context) {
	var req googleLinkStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Invalid(ctx, err)
		return
	}
	accountID := middleware.AccountID(ctx)
	if (req.Password == "") == (req.PhoneCode == "") {
		response.Reject(ctx, http.StatusBadRequest, "invalid_link_proof", "provide exactly one reauthentication method")
		return
	}
	if req.PhoneCode != "" {
		if g.phone == nil {
			response.Reject(ctx, http.StatusForbidden, "phone_reauthentication_unavailable", "phone reauthentication unavailable")
			return
		}
		if err := g.phone.AuthorizePhoneProof(ctx.Request.Context(), accountID, req.PhoneCode); err != nil {
			response.Error(ctx, err)
			return
		}
	} else if err := g.auth.CheckPassword(ctx.Request.Context(), accountID, req.Password); err != nil {
		response.Error(ctx, err)
		return
	}
	authURL, err := g.begin(ctx, "link", accountID, "")
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"authorization_url": authURL})
}

func (g *GoogleHandler) StartLogin(ctx *gin.Context) {
	authURL, err := g.begin(ctx, "login", 0, "")
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.Redirect(http.StatusFound, authURL)
}

// StartPhoneBind starts an OAuth reauthentication, not a Google account link.
// Only the linked Google subject can authorize a phone-binding challenge.
func (g *GoogleHandler) StartPhoneBind(ctx *gin.Context) {
	if g.phone == nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Invalid(ctx, err)
		return
	}
	authURL, err := g.begin(ctx, "phone_bind", middleware.AccountID(ctx), req.Phone)
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"authorization_url": authURL})
}

func googleRandom() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (g *GoogleHandler) stateCookie(ctx *gin.Context, state string, age int) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name: googleOAuthCookie, Value: state, Path: "/users/google/callback", MaxAge: age,
		HttpOnly: true, Secure: g.secureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func (g *GoogleHandler) begin(ctx *gin.Context, purpose string, accountID int64, phone string) (string, error) {
	state, err := googleRandom()
	if err != nil {
		return "", err
	}
	verifier, err := googleRandom()
	if err != nil {
		return "", err
	}
	nonce, err := googleRandom()
	if err != nil {
		return "", err
	}
	record, err := json.Marshal(googlePending{Purpose: purpose, AccountID: accountID, Verifier: verifier, Nonce: nonce, Phone: phone})
	if err != nil {
		return "", err
	}
	if err = g.redis.Set(ctx.Request.Context(), "google:oauth:state:"+state, record, googleOAuthTTL).Err(); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"client_id":             {g.config.GoogleClientID},
		"redirect_uri":          {g.config.GoogleRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(hash[:])},
		"code_challenge_method": {"S256"},
	}
	g.stateCookie(ctx, state, int(googleOAuthTTL.Seconds()))
	ctx.Header("Cache-Control", "no-store")
	return googleAuthorizationURL + "?" + params.Encode(), nil
}
