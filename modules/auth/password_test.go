package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVersionedPasswordHash(t *testing.T) {
	password := strings.Repeat("đ", 40)
	first, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil || first == second {
		t.Fatal("hashes must have independent salts")
	}
	if err := CheckPassword(password, first); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(CheckPassword(password+"x", first), ErrPasswordMismatch) {
		t.Fatal("wrong password accepted")
	}
	if !errors.Is(CheckPassword(password, strings.Replace(first, "19456", "999999999", 1)), ErrPasswordMismatch) {
		t.Fatal("unbounded parameters accepted")
	}
}

func TestLegacyPasswordAndPolicy(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("legacy password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPassword("legacy password", string(hash)); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(CheckPassword(strings.Repeat("a", 73), string(hash)), ErrPasswordMismatch) {
		t.Fatal("legacy bcrypt silently truncated password")
	}
	for _, password := range []string{"short", strings.Repeat("a", 129), string([]byte{0xff})} {
		if !errors.Is(ValidateNewPassword(password), ErrPasswordPolicy) {
			t.Fatal("invalid password policy")
		}
	}
	if !errors.Is(CheckMissingPassword("valid-looking password"), ErrPasswordMismatch) {
		t.Fatal("dummy credential authenticated")
	}
}

func BenchmarkPasswordHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := HashPassword("benchmark only password"); err != nil {
			b.Fatal(err)
		}
	}
}
