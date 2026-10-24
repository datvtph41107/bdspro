package auth

import (
 "context"
 "crypto/rand"
 "crypto/rsa"
 "encoding/base64"
 "encoding/json"
 "math/big"
 "net/http"
 "net/http/httptest"
 "testing"
 "time"

 "github.com/golang-jwt/jwt/v5"
 "github.com/stretchr/testify/require"
)

func TestGoogleIDTokenValidation(t *testing.T) {
 privateKey,err:=rsa.GenerateKey(rand.Reader,2048)
 require.NoError(t,err)
 keys:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
    data:=map[string]any{"keys":[]map[string]string{{
        "kty":"RSA","kid":"unit-key","alg":"RS256","use":"sig",
        "n":base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
        "e":base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
    }}}
    require.NoError(t,json.NewEncoder(w).Encode(data))
 }))
 defer keys.Close()
 makeToken:=func(aud,nonce,issuer string,exp time.Time) string {
    claims:=googleIDClaims{
       RegisteredClaims:jwt.RegisteredClaims{
          Issuer:issuer,Subject:"google-sub-123",Audience:jwt.ClaimStrings{aud},
          IssuedAt:jwt.NewNumericDate(time.Now().Add(-time.Minute)),
          ExpiresAt:jwt.NewNumericDate(exp),
       },
       Nonce:nonce,
    }
    tok:=jwt.NewWithClaims(jwt.SigningMethodRS256,claims)
    tok.Header["kid"]="unit-key"
    signed,err:=tok.SignedString(privateKey)
    require.NoError(t,err)
    return signed
 }
 good:=makeToken("test-client","expected-nonce","https://accounts.google.com",time.Now().Add(time.Hour))
 sub,err:=verifyGoogleIDToken(context.Background(),keys.Client(),keys.URL,good,"test-client","expected-nonce")
 require.NoError(t,err)
 require.Equal(t,"google-sub-123",sub)
 for _,tc:=range []struct{name,raw,aud,nonce string}{
    {"wrong audience",good,"other-client","expected-nonce"},
    {"wrong nonce",good,"test-client","unexpected-nonce"},
    {"wrong issuer",makeToken("test-client","expected-nonce","https://attacker.example",time.Now().Add(time.Hour)),"test-client","expected-nonce"},
    {"expired",makeToken("test-client","expected-nonce","https://accounts.google.com",time.Now().Add(-time.Hour)),"test-client","expected-nonce"},
    {"malformed","not-a-jwt","test-client","expected-nonce"},
 } {
    t.Run(tc.name,func(t *testing.T){
        _,err:=verifyGoogleIDToken(context.Background(),keys.Client(),keys.URL,tc.raw,tc.aud,tc.nonce)
        require.Error(t,err)
    })
 }
}
