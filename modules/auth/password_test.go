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
		if !errors.Is(validateNewPassword(password), ErrPasswordPolicy) {
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

func TestPasswordCompatibilityAndCapacity(t *testing.T) {
	// Independent Argon2id v19 fixture, with the previously persisted PHC format.
	const fixture = "$argon2id$v=19$m=19456,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$cE8/bG8Rg6+LkVzTEsMgLkfdB+qvWau6T9owX+3WGYc"
	const password = "fixture password 123"
	if err := CheckPassword(password, fixture); err != nil {
		t.Fatal("existing Argon2id hash rejected", err)
	}
	hash, err := HashPassword(password)
	if err != nil || !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatal("persisted hash format changed", err)
	}
	for _, p := range []string{strings.Repeat("a", 15), strings.Repeat("a", 128), strings.Repeat("😀", 128)} {
		if err := validateNewPassword(p); err != nil {
			t.Fatal("valid boundary rejected", err)
		}
	}
	for _, p := range []string{strings.Repeat("a", 14), strings.Repeat("a", 129), strings.Repeat("😀", 129)} {
		if err := validateNewPassword(p); !errors.Is(err, ErrPasswordPolicy) {
			t.Fatal("invalid boundary accepted")
		}
	}
	if cap(passwordSlots) != 4 || len(passwordSlots) != 0 {
		t.Fatal("unexpected password capacity or leaked slot")
	}

	// Occupy the actual shared semaphore deterministically; no timing assumptions.
	for i := 0; i < cap(passwordSlots); i++ {
		passwordSlots <- struct{}{}
	}
	defer func() {
		for len(passwordSlots) > 0 {
			<-passwordSlots
		}
	}()
	if _, err := HashPassword(password); !errors.Is(err, ErrPasswordBusy) {
		t.Fatal("hashing bypassed capacity", err)
	}
	if err := CheckPassword(password, fixture); !errors.Is(err, ErrPasswordBusy) {
		t.Fatal("verification bypassed capacity", err)
	}
	if err := CheckMissingPassword(password); !errors.Is(err, ErrPasswordBusy) {
		t.Fatal("dummy verification bypassed capacity", err)
	}
	// Input validation still precedes capacity admission.
	if _, err := HashPassword("short"); !errors.Is(err, ErrPasswordPolicy) {
		t.Fatal("password policy precedence changed", err)
	}
	<-passwordSlots
	if err := CheckPassword(password, fixture); err != nil {
		t.Fatal("released capacity could not be reused", err)
	}
	if len(passwordSlots) != cap(passwordSlots)-1 {
		t.Fatal("verification leaked a slot")
	}
}
