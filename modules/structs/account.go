package structs

import "github.com/datvtph41107/bdspro/modules/optional"

type UpdateProfileOption struct {
	DisplayName optional.Option[string] `json:"display_name"`
	FirstName   optional.Option[string] `json:"first_name"`
	LastName    optional.Option[string] `json:"last_name"`
}

type UpdateAccountSettingsOption struct {
	Locale   optional.Option[string] `json:"locale"`
	Timezone optional.Option[string] `json:"timezone"`
	Currency optional.Option[string] `json:"currency"`
}
