package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mailRecorder struct {
	subject   string
	content   string
	recipient string
}

func (m *mailRecorder) SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error {
	m.subject, m.content = subject, content
	if len(to) == 1 {
		m.recipient = to[0]
	}
	return nil
}

func TestProcessTaskSendVerifyEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	account := db.Account{ID: 11, Email: "person@example.test"}
	mailer := &mailRecorder{}

	store.EXPECT().GetAccountByID(gomock.Any(), account.ID).Return(account, nil)
	store.EXPECT().GetOrCreateVerifyEmailTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, args db.CreateVerifyEmailParams) (db.VerifyEmail, error) {
			require.Equal(t, account.ID, args.AccountID)
			require.Equal(t, account.Email, args.Email)
			require.Len(t, args.SecretCode, 64)
			return db.VerifyEmail{ID: 19, AccountID: account.ID, SecretCode: args.SecretCode}, nil
		},
	)
	processor := &RedisTaskProcessor{store: store, mailer: mailer, verifyEmailURL: "http://localhost:8080/users/verify_email"}
	body, err := json.Marshal(PayloadSendVerifyEmail{AccountID: account.ID})
	require.NoError(t, err)
	task := asynq.NewTask(TaskSendVerifyEmail, body)
	require.NoError(t, processor.ProcessTaskSendVerifyEmail(context.Background(), task))
	require.Equal(t, account.Email, mailer.recipient)
	require.Contains(t, mailer.content, "email_id=19")
	require.Contains(t, mailer.content, "secret_code=")
}

func TestProcessTaskBadPayload(t *testing.T) {
	processor := &RedisTaskProcessor{}
	err := processor.ProcessTaskSendVerifyEmail(context.Background(), asynq.NewTask(TaskSendVerifyEmail, []byte("{invalid")))
	require.Error(t, err)
	require.True(t, errors.Is(err, asynq.SkipRetry))
}

func TestProcessTaskSendVerifyEmailReusesVerificationOnRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	account := db.Account{ID: 37, Email: "retry@example.test"}
	record := db.VerifyEmail{ID: 91, AccountID: account.ID, Email: account.Email, SecretCode: "same-secret"}

	store.EXPECT().GetAccountByID(gomock.Any(), account.ID).Return(account, nil).Times(2)
	store.EXPECT().GetOrCreateVerifyEmailTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.CreateVerifyEmailParams) (db.VerifyEmail, error) {
			require.Equal(t, account.ID, arg.AccountID)
			return record, nil
		},
	).Times(2)

	mailer := &mailRecorder{}
	processor := &RedisTaskProcessor{store: store, mailer: mailer, verifyEmailURL: "http://localhost:8080/users/verify_email"}
	body, err := json.Marshal(PayloadSendVerifyEmail{AccountID: account.ID})
	require.NoError(t, err)
	task := asynq.NewTask(TaskSendVerifyEmail, body)
	for i := 0; i < 2; i++ {
		require.NoError(t, processor.ProcessTaskSendVerifyEmail(context.Background(), task))
		require.Contains(t, mailer.content, "email_id=91")
		require.Contains(t, mailer.content, "same-secret")
	}
}

func TestProcessTaskSkipsAlreadyVerifiedAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	account := db.Account{ID: 38, Email: "done@example.test", IsEmailVerified: true}
	store.EXPECT().GetAccountByID(gomock.Any(), account.ID).Return(account, nil)
	processor := &RedisTaskProcessor{store: store, mailer: &mailRecorder{}}
	body, err := json.Marshal(PayloadSendVerifyEmail{AccountID: account.ID})
	require.NoError(t, err)
	require.NoError(t, processor.ProcessTaskSendVerifyEmail(
		context.Background(), asynq.NewTask(TaskSendVerifyEmail, body),
	))
}
