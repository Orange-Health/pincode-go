# pincode

Looks up the city, state and ISO 3166-2 code for an Indian pincode.

The pincode data is part of the library (`data/pincode_city_state_ranges.csv`, compiled into
your binary with `go:embed`). There is nothing to load or configure: the library parses the
data once on the first lookup and keeps it in memory, indexed by the first 3 digits of the
pincode. Each lookup then takes about 80 ns.

```go
import "github.com/orangehealth/pincode"

loc, err := pincode.Lookup("560105")
switch {
case errors.Is(err, pincode.ErrInvalid):  // not a 6-digit pincode
case errors.Is(err, pincode.ErrNotFound): // valid, but not in the data
default:
	fmt.Println(loc.City, loc.State, loc.ISO3166SubDivisionCode) // Bengaluru Karnataka IN-KA
}
```

- `Lookup` accepts "560105" or "560 105" and is safe to call from any goroutine.
- The first call parses the data (~15 ms). To avoid that on a request path, call `pincode.Lookup("110001")` once at startup.

## Updating the data

Replace `data/pincode_city_state_ranges.csv`, run `go test ./...`, and release a new version.
The tests validate the shipped file (bad numbers, overlapping ranges, or a range that crosses a
3-digit prefix fail the build), so a broken data file can't reach production.

See [DESIGN.md](DESIGN.md) for internals and data caveats. Benchmarks: `scripts/bench.sh`.

CSV header: `pincode_from,pincode_till,city,state,iso3166_subdivision_code`
