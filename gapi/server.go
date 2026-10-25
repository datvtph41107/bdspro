package gapi

import (
	"fmt"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
)

// Server provides BDSPro RPC handlers; listeners belong in main.go.
type Server struct {
	pb.UnimplementedBDSProServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	verificationScheduler auth.VerificationTaskScheduler
}

func NewServer(config util.Config, store db.Store, distributor worker.TaskDistributor) (*Server, error) {
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	return &Server{
		config: config,
		store: store,
		tokenMaker: maker,
		verificationScheduler: worker.NewVerificationScheduler(distributor),
	}, nil
}
