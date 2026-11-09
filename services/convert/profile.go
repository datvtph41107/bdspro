package convert

import (
	account_model "github.com/datvtph41107/bdspro/models/account"
	"github.com/datvtph41107/bdspro/modules/structs"
)

func ToAccountSettings(p *account_model.Profile) structs.AccountSettingsResponse {
	return structs.AccountSettingsResponse{Locale: p.Locale, Timezone: p.Timezone, Currency: p.Currency}
}

func ToAccountProfile(p *account_model.Profile) structs.AccountProfileResponse {
	return structs.AccountProfileResponse{
		AccountID: p.AccountID, PersonID: p.PersonID,
		Email: p.Email, IsEmailVerified: p.IsEmailVerified, CreatedAt: p.CreatedAt,
		Profile: structs.PersonProfileResponse{
			DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName,
		},
		Security: structs.AccountSecurityResponse{
			HasPassword: p.HasPassword, HasPhone: p.HasPhone, HasOAuth2: p.HasOAuth2,
		},
		Settings: ToAccountSettings(p),
	}
}
