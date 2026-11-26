package auth

import (
	"net/url"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestGoogleStepUp(t *testing.T) {
	base := GoogleAuthorizationURL(setting.GoogleSettings{ClientID: "test"}, "state", "nonce", "verifier")
	raw, err := GoogleStepUpURL(base)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Query().Get("claims") != `{"id_token":{"auth_time":{"essential":true}}}` ||
		u.Query().Get("state") != "state" || u.Query().Get("nonce") != "nonce" ||
		u.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("step-up must preserve authorization binding")
	}
	now := time.Unix(1800000000, 0)
	for _, tc := range []struct {
		at   int64
		want bool
	}{
		{0, false}, {now.Unix() - 301, false}, {now.Unix() - 300, false},
		{now.Unix() - 299, true}, {now.Unix(), true}, {now.Unix() + 30, true}, {now.Unix() + 31, false},
	} {
		if GoogleAuthTimeRecent(tc.at, now) != tc.want {
			t.Fatal("incorrect auth_time freshness")
		}
	}
}
