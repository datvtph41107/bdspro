package db

// Account is the stable application model. SQLC emits query-specific row
// structures for nullable email projections (COALESCE), so conversion is
// explicit rather than changing generated files or fabricating an email.
func AccountFromIDRow(row GetAccountByIDRow) Account {
	return Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}
}

func AccountFromEmailRow(row GetAccountByEmailRow) Account {
	return Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}
}
