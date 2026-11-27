package phonedelivery

import (
	"context"
	"net"
	"strings"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/mail"
	"github.com/datvtph41107/bdspro/modules/setting"
)

type mailpitSender struct{ channel Channel }

// Send captures an OTP in the local Mailpit inbox only. Recheck the development
// boundary here as well as at configuration loading and service construction.
func (s mailpitSender) Send(ctx context.Context, m Message) (Receipt, error) {
	channel := string(s.channel)
	canonical, err := token.NormalizePhone(m.Phone, "")
	host, _, addressErr := net.SplitHostPort(setting.Mailer.Address)
	ip := net.ParseIP(host)
	local := host == "localhost" || host == "mailpit" || ip != nil && ip.IsLoopback()
	if setting.Common.RunMode != "development" || setting.Mailer.Protocol != "smtp" ||
		addressErr != nil || !local || err != nil || canonical != m.Phone || !token.ValidPhoneCode(m.Code) ||
		channel != "sms" && channel != "zalo" && channel != "telegram" {
		return Receipt{}, &Failure{Reason: "mailpit_unavailable"}
	}
	destination := channel + "." + strings.TrimPrefix(m.Phone, "+") + "@phone.invalid"
	err = mail.SendPlain(ctx, destination, "[DEV "+channel+"] BDSPro phone verification",
		"Development delivery only.\r\nChannel: "+channel+"\r\nPhone: "+m.Phone+"\r\nVerification code: "+m.Code+"\r\nExpires in 5 minutes.\r\n")
	if err != nil {
		return Receipt{}, &Failure{Reason: "mailpit_unavailable"}
	}
	return Receipt{Provider: "mailpit"}, nil
}
