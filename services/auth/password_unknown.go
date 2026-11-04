package auth

import authcrypto "github.com/datvtph41107/bdspro/modules/auth"

// A fixed bcrypt hash at the same cost as freshly issued password hashes.
// It belongs to no account and is deliberately not a credential or a secret.
// Missing-account and passwordless-account login attempts must still pay
// the normal bcrypt verification cost before returning InvalidCredentials.
const dummyPasswordHash = "$2a$10$jf3HO8nLz6D6fJUAiJ7sB.wHyxxdZneCFqfIxSyWP4t9wbygi36Ya"

func checkUnknownPassword(password string) {
	_ = authcrypto.CheckPassword(password, dummyPasswordHash)
}
