package templates

import (
    "bytes"
    "strings"
    "testing"
)
func TestVerifyEmailPageEscapesToken(t *testing.T) {
    var out bytes.Buffer
    if err:=VerifyEmailPage.Execute(&out,struct {
        EmailID int64
        SecretCode string
    }{EmailID:42,SecretCode:`"><script>alert(1)</script>`});err!=nil {t.Fatal(err)}
    html:=out.String()
    if strings.Contains(html,"<script>"){t.Fatalf("script was not escaped: %s",html)}
    if !strings.Contains(html,`name="email_id" value="42"`){t.Fatalf("missing form field: %s",html)}
}
func TestPasswordResetPageEscapesToken(t *testing.T) {
    var out bytes.Buffer
    if err:=PasswordResetPage.Execute(&out,`"><script>alert(1)</script>`);err!=nil {t.Fatal(err)}
    if strings.Contains(out.String(),"<script>"){t.Fatal("unescaped token")}
}
