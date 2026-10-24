package db

import (
    "context"
    "crypto/hmac"
    "errors"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var (
    ErrOTPInvalid = errors.New("invalid or expired phone code")
    ErrOTPCooldown = errors.New("phone code temporarily unavailable")
)

// PhoneOTPRepository is deliberately independent from sqlc's generated Store.
type PhoneOTPRepository interface {
    AccountIDForPhone(ctx context.Context, phone string) (int64,error)
    StartChallenge(ctx context.Context, phone,purpose string,accountID int64, mac []byte,devCode string) error
    ConsumeChallenge(ctx context.Context, phone,purpose string,accountID int64,mac []byte) error
}

type phoneOTPStore struct { pool *pgxpool.Pool }
func NewPhoneOTPStore(pool *pgxpool.Pool) PhoneOTPRepository {return &phoneOTPStore{pool:pool}}

func (s *phoneOTPStore) AccountIDForPhone(ctx context.Context, phone string) (int64,error) {
    var accountID int64
    err:=s.pool.QueryRow(ctx,`SELECT account_id FROM phone_credentials WHERE phone_e164=$1`,phone).Scan(&accountID)
    return accountID,err
}

// Atomic UPSERT preserves the failed-attempt window when a code is resent.
func (s *phoneOTPStore) StartChallenge(ctx context.Context,phone,purpose string,accountID int64,mac []byte,devCode string) error {
    tag,err:=s.pool.Exec(ctx,`
INSERT INTO phone_otp_challenges
  (phone_e164,purpose,account_id,code_mac,dev_code,expires_at,next_send_at,
   failed_attempts,attempt_window_started_at,locked_until)
VALUES ($1,$2,$3,$4,NULLIF($5,''),now()+interval '5 minutes',
        now()+interval '60 seconds',0,now(),'epoch'::timestamptz)
ON CONFLICT (phone_e164,purpose) DO UPDATE
SET account_id=EXCLUDED.account_id,
    code_mac=EXCLUDED.code_mac,
    dev_code=EXCLUDED.dev_code,
    expires_at=EXCLUDED.expires_at,
    next_send_at=EXCLUDED.next_send_at,
    failed_attempts=CASE WHEN phone_otp_challenges.attempt_window_started_at <= now()-interval '15 minutes'
                         THEN 0 ELSE phone_otp_challenges.failed_attempts END,
    attempt_window_started_at=CASE WHEN phone_otp_challenges.attempt_window_started_at <= now()-interval '15 minutes'
                                   THEN now() ELSE phone_otp_challenges.attempt_window_started_at END,
    locked_until=CASE WHEN phone_otp_challenges.attempt_window_started_at <= now()-interval '15 minutes'
                     THEN 'epoch'::timestamptz ELSE phone_otp_challenges.locked_until END
WHERE phone_otp_challenges.next_send_at<=now()
  AND phone_otp_challenges.locked_until<=now()
  AND (phone_otp_challenges.failed_attempts<5
       OR phone_otp_challenges.attempt_window_started_at <= now()-interval '15 minutes')
`,phone,purpose,accountID,mac,devCode)
    if err!=nil{return err}
    if tag.RowsAffected()!=1{return ErrOTPCooldown}
    return nil
}

// FOR UPDATE serializes attempts; a successful challenge is consumed with
// phone binding (or credential recheck) in the same transaction.
func (s *phoneOTPStore) ConsumeChallenge(ctx context.Context,phone,purpose string,accountID int64,submittedMAC []byte) error {
    tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
    if err!=nil{return err}
    defer tx.Rollback(ctx)

    var savedAccount int64
    var expectedMAC []byte
    var expiresAt,lockedUntil time.Time
    var failed int
    err=tx.QueryRow(ctx,`SELECT account_id,code_mac,expires_at,failed_attempts,locked_until
        FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose=$2 FOR UPDATE`,phone,purpose).
        Scan(&savedAccount,&expectedMAC,&expiresAt,&failed,&lockedUntil)
    if errors.Is(err,pgx.ErrNoRows){return ErrOTPInvalid}
    if err!=nil{return err}
    now:=time.Now()
    if savedAccount!=accountID || !now.Before(expiresAt) || now.Before(lockedUntil) || failed>=5 {
        return ErrOTPInvalid
    }
    if !hmac.Equal(expectedMAC,submittedMAC) {
        _,err=tx.Exec(ctx,`UPDATE phone_otp_challenges
            SET failed_attempts=failed_attempts+1,
                locked_until=CASE WHEN failed_attempts+1>=5 THEN now()+interval '15 minutes'
                                  ELSE locked_until END
            WHERE phone_e164=$1 AND purpose=$2`,phone,purpose)
        if err!=nil{return err}
        if err=tx.Commit(ctx);err!=nil{return err}
        return ErrOTPInvalid
    }
    switch purpose {
    case "bind":
        _,err=tx.Exec(ctx,`INSERT INTO phone_credentials(account_id,phone_e164,verified_at)
            VALUES ($1,$2,now()) ON CONFLICT (account_id) DO UPDATE
            SET phone_e164=EXCLUDED.phone_e164,verified_at=EXCLUDED.verified_at`,accountID,phone)
        if err!=nil{return err}
    case "login":
        var current int64
        err=tx.QueryRow(ctx,`SELECT account_id FROM phone_credentials WHERE phone_e164=$1 FOR SHARE`,phone).Scan(&current)
        if errors.Is(err,pgx.ErrNoRows){return ErrOTPInvalid}
        if err!=nil{return err}
        if current!=accountID{return ErrOTPInvalid}
    default:
        return ErrOTPInvalid
    }
    if _,err=tx.Exec(ctx,`DELETE FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose=$2`,phone,purpose);err!=nil{return err}
    return tx.Commit(ctx)
}

var _ PhoneOTPRepository=(*phoneOTPStore)(nil)
