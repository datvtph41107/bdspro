package server

import (
	"errors"

	authcontrollers "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	middleware "github.com/datvtph41107/bdspro/app/http/middleware"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/routes"
)

// EnablePhoneOTP wires a development-only controller; the canonical SQLStore
// is the production OTP repository and shares the main PostgreSQL pool.
func (server *Server) EnablePhoneOTP(repository db.PhoneOTPRepository) error {
	if server.config.Environment != "development" {
		return nil
	}
	if repository == nil {
		return errors.New("development phone OTP requires repository")
	}
	server.phoneOTP = repository
	controller := authcontrollers.NewPhoneController(
		server.store, repository, server.tokenMaker, server.config,
	)
	routes.MountPhone(server.router, routes.PhoneHandlers{
		StartLogin: controller.StartLogin, VerifyLogin: controller.VerifyLogin,
		StartBind: controller.StartBind, VerifyBind: controller.VerifyBind,
		PreStart: middleware.PrePhoneLoginStart(), PreVerify: middleware.PrePhoneVerify(),
		PreBind: middleware.PrePhoneBind(), SufLogin: SufLogin(),
		Authenticate: Authenticate(server.tokenMaker),
		Source:       server.IPActiveSource, Identity: server.IPActive,
		StartIdentity:  middleware.PhoneStartIdentity,
		VerifyIdentity: middleware.PhoneVerifyIdentity,
		BindIdentity:   middleware.PhoneBindIdentity,
	})
	return nil
}
