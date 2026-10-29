package jobs

import (
    "context"
    "crypto/sha256"
    "errors"
    "regexp"
    "testing"

    db "github.com/datvtph41107/bdspro/internal/database"
)

type passwordResetStoreFake struct {
    account db.Account
    accountErr error
    insertErr error
    hashes [][]byte
}

func (s *passwordResetStoreFake) GetAccountByID(_ context.Context, _ int64) (db.Account,error) {
    return s.account,s.accountErr
}
func (s *passwordResetStoreFake) CreatePasswordResetToken(_ context.Context,arg db.CreatePasswordResetTokenParams) error {
    if s.insertErr!=nil {return s.insertErr}
    s.hashes=append(s.hashes,append([]byte(nil),arg.TokenHash...))
    return nil
}
func TestPasswordResetJobGeneratesProofInWorkerAndRetriesSMTP(t *testing.T) {
    ctx:=context.Background()
    store:=&passwordResetStoreFake{account:db.Account{ID:42,Email:"verified@example.test",IsEmailVerified:true}}
    sender:=&fakeMailSender{fail:errors.New("SMTP unavailable")}
    url:="https://example.test/users/verify_email"
    if err:=SendPasswordResetEmail(ctx,store,sender,url,42);err==nil {t.Fatal("SMTP failure must be retryable")}
    if len(store.hashes)!=1 || len(sender.content)!=0 {t.Fatal("failed SMTP did not persist token hash")}
    sender.fail=nil
    if err:=SendPasswordResetEmail(ctx,store,sender,url,42);err!=nil {t.Fatal(err)}
    if len(sender.content)!=1 || len(store.hashes)!=2 {t.Fatal("retry failed to issue usable proof")}
    match:=regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(sender.content[0])
    if len(match)!=2 {t.Fatal("email missing one-time link")}
    digest:=sha256.Sum256([]byte(match[1]))
    for i:=range digest {if store.hashes[1][i]!=digest[i] {t.Fatal("database persisted wrong hash")}}
    if len(store.hashes[1])!=32 {t.Fatal("reset hash must be 32 bytes")}
}
func TestPasswordResetJobSkipsUnverifiedAccounts(t *testing.T) {
    store:=&passwordResetStoreFake{account:db.Account{ID:9,Email:"unverified@example.test"}}
    sender:=&fakeMailSender{}
    if err:=SendPasswordResetEmail(context.Background(),store,sender,"https://example.test/users/verify_email",9);err!=nil {t.Fatal(err)}
    if len(store.hashes)!=0 || len(sender.content)!=0 {t.Fatal("unverified email must not receive reset token")}
}
func TestPasswordResetJobPropagatesDatabaseFailure(t *testing.T) {
    boom:=errors.New("postgres unavailable")
    store:=&passwordResetStoreFake{account:db.Account{ID:5,Email:"verified@example.test",IsEmailVerified:true},insertErr:boom}
    sender:=&fakeMailSender{}
    if err:=SendPasswordResetEmail(context.Background(),store,sender,"https://example.test/users/verify_email",5);!errors.Is(err,boom) {t.Fatalf("database failure not retryable: %v",err)}
    if len(sender.content)!=0 {t.Fatal("no email before DB commit")}
}
