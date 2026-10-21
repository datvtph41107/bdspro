package util

import (
	"errors"
	"testing"
)

func TestDecimalString(t *testing.T) {
	cases := []struct {
		name, in, want    string
		integer, fraction int
		valid             bool
	}{
		{"integer", "000123", "123", 14, 4, true},
		{"fraction", "000.0100", "0.01", 14, 4, true},
		{"spaces", " 042.1000 ", "42.1", 14, 4, true},
		{"leading zero", "0.01", "0.01", 14, 4, true},
		{"zero", "000.0000", "", 14, 4, false},
		{"negative", "-1", "", 14, 4, false},
		{"exponent", "1e3", "", 14, 4, false},
		{"empty fraction", "1.", "", 14, 4, false},
		{"too precise", "1.00001", "", 14, 4, false},
		{"too many whole digits", "123456789012345", "", 14, 4, false},
		{"huge but valid price", "123456789012345.1234", "123456789012345.1234", 15, 4, true},
		{"spaces inside", "1 .4", "", 14, 4, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecimalString(tc.in, tc.integer, tc.fraction)
			if tc.valid {
				if err != nil || got != tc.want {
					t.Fatalf("got %q, %v; want %q", got, err, tc.want)
				}
				return
			}
			if !errors.Is(err, ErrInvalidDecimal) {
				t.Fatalf("expected invalid decimal, got %q, %v", got, err)
			}
		})
	}
}

func TestTimestampUTC(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		valid       bool
	}{
		{"", "", true},
		{"  ", "", true},
		{"2026-10-21T10:30:00+07:00", "2026-10-21T03:30:00Z", true},
		{"2026-10-21T03:30:00Z", "2026-10-21T03:30:00Z", true},
		{"tomorrow", "", false},
	} {
		got, err := TimestampUTC(tc.input)
		if tc.valid && (err != nil || got != tc.want) {
			t.Fatalf("input=%q got=%q err=%v", tc.input, got, err)
		}
		if !tc.valid && !errors.Is(err, ErrInvalidTimestamp) {
			t.Fatalf("invalid input=%q err=%v", tc.input, err)
		}
	}
}

func TestCurrencyCode(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		valid       bool
	}{
		{"vnd", "VND", true}, {" usd ", "USD", true}, {"1US", "", false}, {"VN", "", false}, {"☃☃☃", "", false},
	} {
		got, err := CurrencyCode(tc.input)
		if tc.valid && (err != nil || got != tc.want) {
			t.Fatalf("got=%q err=%v", got, err)
		}
		if !tc.valid && !errors.Is(err, ErrInvalidCurrency) {
			t.Fatalf("expected invalid currency %q: %v", tc.input, err)
		}
	}
}
