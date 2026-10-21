package gapi

import (
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertAccount(account db.Account) *pb.Account {
	return &pb.Account{
		AccountId: account.ID, Email: account.Email,
		CreatedAt: timestamppb.New(account.CreatedAt), IsEmailVerified: account.IsEmailVerified,
	}
}

