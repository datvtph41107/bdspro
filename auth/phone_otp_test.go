package auth

import (
    "crypto/hmac"
    "strings"
    "testing"

    "github.com/datvtph41107/bdspro/util"
    "github.com/stretchr/testify/require"
)

func TestPhoneOTPPurposeBinding(t *testing.T) {
    s:=PhoneOTP{Config:util.Config{Environment:"development",TokenSymmetricKey:strings.Repeat("s",32)}}
    code,err:=s.generateCode()
    require.NoError(t,err)
    require.Regexp(t,`^[0-9]{6}$`,code)
    digest:=s.mac("+84901234567","bind",7,code)
    require.Len(t,digest,32)
    require.True(t,hmac.Equal(digest,s.mac("+84901234567","bind",7,code)))
    require.False(t,hmac.Equal(digest,s.mac("+84901234567","login",7,code)))
    require.False(t,hmac.Equal(digest,s.mac("+84901234567","bind",8,code)))
    require.False(t,hmac.Equal(digest,s.mac("+84901234568","bind",7,code)))
}

func TestPhoneOTPDisabledOutsideDevelopment(t *testing.T) {
    s:=PhoneOTP{Config:util.Config{Environment:"production",TokenSymmetricKey:strings.Repeat("s",32)}}
    require.Error(t,s.allowed())
}
