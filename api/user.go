package api

import (
    "github.com/datvtph41107/bdspro/auth"
    authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
    "github.com/datvtph41107/bdspro/app/http/resources"
    db "github.com/datvtph41107/bdspro/db/sqlc"
)

// Compatibility types while Google/Phone HTTP handlers are moved next.
type createUserRequest = authrequest.RegisterBody
type loginUserRequest = authrequest.LoginBody
type userResponse = resources.User
type loginUserResponse = resources.Login

func newUserResponse(account db.Account) userResponse { return resources.NewUser(account) }
func newLoginUserResponse(result auth.LoginResult) loginUserResponse {
    return resources.NewLogin(result)
}
