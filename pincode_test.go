package pincode

import (
	"errors"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	cases := map[string]Location{
		"560105":  {"Bengaluru", "Karnataka", "IN-KA"},
		"560 001": {"Bengaluru", "Karnataka", "IN-KA"},
		"110001":  {"New Delhi", "Delhi", "IN-DL"},
		"500032":  {"Hyderabad", "Telangana", "IN-TS"},
		"682001":  {"Kochi", "Kerala", "IN-KL"},
		"110043":  {"New Delhi", "Delhi", "IN-DL"},
		"403601":  {"Margao", "Goa", "IN-GA"},
		"791110":  {"Itanagar", "Arunachal Pradesh", "IN-AR"},
	}
	for pin, want := range cases {
		if got, err := Lookup(pin); err != nil || got != want {
			t.Errorf("%s: got %+v, %v; want %+v", pin, got, err, want)
		}
	}
}

func TestErrors(t *testing.T) {
	for _, p := range []string{"", "12345", "abcdef", "0560105", "099999"} {
		if _, err := Lookup(p); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q: want ErrInvalid, got %v", p, err)
		}
	}
	if _, err := Lookup("100000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("100000: want ErrNotFound, got %v", err)
	}
}

func TestEmbeddedDataIsValid(t *testing.T) {
	// load the shipped CSV directly so a bad data file fails the tests, not production
	if err := load(strings.NewReader(embeddedCSV)); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, b := range buckets {
		if b != nil {
			n += len(b.entries)
		}
	}
	if n < 10000 {
		t.Fatalf("only %d ranges loaded", n)
	}
}

func TestConcurrentFirstUse(t *testing.T) {
	done := make(chan struct{})
	for i := 0; i < 16; i++ {
		go func() { Lookup("110001"); done <- struct{}{} }()
	}
	for i := 0; i < 16; i++ {
		<-done
	}
}

func TestOverlapRejected(t *testing.T) {
	csv := "pincode_from,pincode_till,city,state,iso3166_subdivision_code\n" +
		"560001,560100,A,X,IN-KA\n560050,560200,B,X,IN-KA\n"
	ensureLoaded()
	saved := buckets
	if err := load(strings.NewReader(csv)); err == nil {
		t.Fatal("expected overlap error")
	}
	if buckets != saved {
		t.Fatal("failed load must not replace loaded data")
	}
}

func TestPrefixCrossRejected(t *testing.T) {
	ensureLoaded()
	csv := "pincode_from,pincode_till,city,state,iso3166_subdivision_code\n560900,561100,A,X,IN-KA\n"
	if err := load(strings.NewReader(csv)); err == nil {
		t.Fatal("expected prefix-crossing error")
	}
}
