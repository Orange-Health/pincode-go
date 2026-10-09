// Package pincode resolves Indian pincodes to city, state and ISO 3166-2 subdivision code.
//
// The pincode range data ships inside the library (data/pincode_city_state_ranges.csv,
// compiled in with go:embed). Callers never load anything; just call Lookup:
//
//	loc, err := pincode.Lookup("560105") // {Bengaluru Karnataka IN-KA}
//
// The data is parsed once, on the first Lookup, and is read-only afterwards,
// so Lookup is safe to call from any goroutine.
package pincode

import (
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Location is the lookup result.
type Location struct {
	City                   string `json:"City"`
	State                  string `json:"State"`
	ISO3166SubDivisionCode string `json:"ISO3166SubDivisionCode"`
}

var (
	ErrInvalid  = errors.New("pincode: not a valid 6-digit pincode")
	ErrNotFound = errors.New("pincode: pincode not found")
)

//go:embed data/pincode_city_state_ranges.csv
var embeddedCSV string

type rangeEntry struct {
	from, till int
	loc        Location
}

// bucket holds the ranges whose pincodes share the same first 3 digits, sorted by start.
type bucket struct {
	starts  []int
	entries []rangeEntry
}

var (
	once    sync.Once
	buckets [900]*bucket // index = first 3 digits - 100; nil = no data for that prefix
)

// ensureLoaded parses the embedded data exactly once. The data is validated by the
// package tests, so a failure here means a broken build; it panics with the reason.
func ensureLoaded() {
	once.Do(func() {
		if err := load(strings.NewReader(embeddedCSV)); err != nil {
			panic(err)
		}
	})
}

// Lookup returns the city, state and ISO 3166-2 code for a pincode.
// Accepts "560105" or "560 105".
func Lookup(pin string) (Location, error) {
	ensureLoaded()
	pin = strings.ReplaceAll(strings.TrimSpace(pin), " ", "")
	if len(pin) != 6 || pin[0] == '0' {
		return Location{}, ErrInvalid
	}
	n, err := strconv.Atoi(pin)
	if err != nil {
		return Location{}, ErrInvalid
	}
	b := buckets[n/1000-100]
	if b == nil {
		return Location{}, ErrNotFound
	}
	i := sort.SearchInts(b.starts, n+1) - 1 // last range in this prefix starting at or before n
	if i < 0 || n > b.entries[i].till {
		return Location{}, ErrNotFound
	}
	return b.entries[i].loc, nil
}

func load(r io.Reader) error {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return fmt.Errorf("pincode: reading header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))] = i
	}
	for _, n := range []string{"pincode_from", "pincode_till", "city", "state", "iso3166_subdivision_code"} {
		if _, ok := col[n]; !ok {
			return fmt.Errorf("pincode: missing column %q", n)
		}
	}

	var es []rangeEntry
	for line := 2; ; line++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("pincode: line %d: %w", line, err)
		}
		from, err1 := strconv.Atoi(strings.TrimSpace(rec[col["pincode_from"]]))
		till, err2 := strconv.Atoi(strings.TrimSpace(rec[col["pincode_till"]]))
		if err1 != nil || err2 != nil || from > till || from < 100000 || till > 999999 {
			return fmt.Errorf("pincode: line %d: bad range", line)
		}
		if from/1000 != till/1000 {
			return fmt.Errorf("pincode: line %d: range %d-%d crosses a 3-digit prefix; split it", line, from, till)
		}
		es = append(es, rangeEntry{from, till, Location{
			City:                   strings.TrimSpace(rec[col["city"]]),
			State:                  strings.TrimSpace(rec[col["state"]]),
			ISO3166SubDivisionCode: strings.TrimSpace(rec[col["iso3166_subdivision_code"]]),
		}})
	}
	if len(es) == 0 {
		return errors.New("pincode: no rows in CSV")
	}

	sort.Slice(es, func(i, j int) bool { return es[i].from < es[j].from })
	var bs [900]*bucket
	for i, e := range es {
		if i > 0 && e.from <= es[i-1].till {
			return fmt.Errorf("pincode: overlapping ranges %d-%d and %d-%d",
				es[i-1].from, es[i-1].till, e.from, e.till)
		}
		k := e.from/1000 - 100
		if bs[k] == nil {
			bs[k] = &bucket{}
		}
		bs[k].starts = append(bs[k].starts, e.from)
		bs[k].entries = append(bs[k].entries, e)
	}
	buckets = bs // publish only after full validation
	return nil
}
