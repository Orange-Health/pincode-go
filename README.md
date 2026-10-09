# pincode

A Go library that looks up the city, state and ISO 3166-2 code for an Indian pincode.

```go
pincode import "github.com/orangehealth/pincode-go"

loc, err := pincode.Lookup("560105")
switch {
case errors.Is(err, pincode.ErrInvalid):  // not a 6-digit pincode
case errors.Is(err, pincode.ErrNotFound): // valid, but not in the data
default:
	fmt.Println(loc.City, loc.State, loc.ISO3166SubDivisionCode) // Bengaluru Karnataka IN-KA
}
```

- **Nothing to load or configure.** The pincode data ships inside the library (`data/pincode_city_state_ranges.csv`, compiled in with `go:embed`).
- **Fast.** The first `Lookup` parses the data once (~15 ms) and keeps it in memory (1.9 MB). Every lookup after that takes ~80 ns, with zero allocations.
- **Safe for concurrent use.** `Lookup` can be called from any goroutine. It accepts "560105" or "560 105".
- To keep the one-time parse off a request path, call `pincode.Lookup("110001")` once at startup.

## What "city" means

- **Towns:** the town the pincode belongs to.
- **Villages:** the sub-district headquarters town (taluk, tehsil, mandal, block or circle).
- **Metros and state capitals:** the city name, not the neighbourhood (e.g. Karol Bagh → New Delhi, Kukatpally → Hyderabad).

See [DESIGN.md](DESIGN.md) for internals, design choices and **data caveats**.

## Updating the data

1. Replace `data/pincode_city_state_ranges.csv`. The header must be `pincode_from,pincode_till,city,state,iso3166_subdivision_code`.
2. Run `go test ./...`. The tests reject bad rows, overlapping ranges, and ranges that cross a 3-digit prefix, so a broken file can't be released.
3. Run `scripts/bench.sh -compare bench/results/baseline.txt` to check speed against the baseline.
4. Release a new version.

## Benchmarks

```sh
scripts/bench.sh                                     # run and save to bench/results/
scripts/bench.sh -compare bench/results/baseline.txt # compare with the baseline (benchstat)
```

## Data sources and attribution

The pincode-to-city mapping in `data/` was derived from the following sources. If you redistribute this library or its data, keep this section.

| Source | Used for | Licence |
|---|---|---|
| **India Post — All India Pincode Directory**, published on the [Open Government Data Platform India (data.gov.in)](https://www.data.gov.in/) (Dec 2021 release) | Pincode list, post offices, districts, states | [Government Open Data License – India](https://www.data.gov.in/government-open-data-license-india) |
| **Pincode boundaries**, data.gov.in | Locating each pincode's area | Government Open Data License – India |
| **Local Government Directory (LGD) sub-district boundaries**, Ministry of Panchayati Raj / Bharatmaps | Sub-district (taluk / tehsil / mandal / block) and HQ resolution | Compiled as CC0 1.0 (see below) |
| **Urban local body boundaries**, Swachh Bharat Mission (Urban), MoHUA; West Bengal AMRUT | Municipal boundaries of towns and cities | Compiled as CC0 1.0 (see below) |
| **[ramSeraph/indian_admin_boundaries](https://github.com/ramSeraph/indian_admin_boundaries)**, with credit to [Datameet](https://datameet.org/) | Compiled GeoJSON of the three boundary datasets above | [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/); attribution to Datameet and the original government sources requested |
| **[GeoNames](https://www.geonames.org/)** (via the [all-the-cities](https://github.com/zeke/all-the-cities) package) | Town names, locations and populations | [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/); **attribution required** |
| **ISO 3166-2:IN** | Subdivision codes (e.g. IN-KA, IN-TS) | Code list (ISO) |

Contains data from GeoNames (geonames.org), licensed under CC BY 4.0. Contains data from the Open Government Data Platform India (data.gov.in), licensed under the Government Open Data License – India. Boundary data from Datameet and ramSeraph/indian_admin_boundaries (CC0 1.0), originally published by the Government of India.

City assignments, metro rollups and manual overrides are derived work by Orange Health. They are provided as-is, without warranty of accuracy; see the caveats in [DESIGN.md](DESIGN.md).