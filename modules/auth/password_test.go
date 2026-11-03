package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
    hash, err := HashPassword("my-safe-password")
    if err != nil { t.Fatal(err) }
    if hash == "my-safe-password" { t.Fatal("stored plaintext password") }
    if err := CheckPassword("my-safe-password",hash);err!=nil {t.Fatal(err)}
    if err := CheckPassword("incorrect",hash);err==nil {t.Fatal("invalid password accepted")}
}
