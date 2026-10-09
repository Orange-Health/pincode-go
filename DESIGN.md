# pincode — Design

## 1. Purpose

Resolve an Indian pincode to `{City, State, ISO3166SubDivisionCode}` (e.g. `560105` → `Bengaluru, Karnataka, IN-KA`) in-process, with no database or network call.

## 2. API

```go
func Lookup(pin string) (Location, error)

type Location struct {
    City                   string
    State                  string
    ISO3166SubDivisionCode string   // e.g. "IN-KA"
}

var ErrInvalid  // input is not a 6-digit pincode ("560105" and "560 105" are accepted)
var ErrNotFound // valid pincode, not in the data
```

`Lookup` is the only entry point. Callers never load, configure or ship data.

## 3. How it works

| Stage | Design |
|---|---|
| Packaging | `data/pincode_city_state_ranges.csv` is compiled into the binary with `go:embed`. The data and the code are versioned together. |
| Loading | Lazy. The first `Lookup` parses the CSV once, guarded by `sync.Once` (~15 ms). Later calls do no further loading. |
| Index | A fixed array of 900 buckets indexed by the first 3 digits (`pin/1000 − 100`). Each bucket holds its ranges sorted by `pincode_from`. |
| Lookup | One array read selects the bucket, then a binary search within it (≤131 ranges, ≤8 comparisons) finds the range, then a check that `pin ≤ pincode_till`. |
| Concurrency | Data is immutable after loading, so `Lookup` is safe from any goroutine without locks. |
| Failure | The shipped CSV is validated by `go test` (bad rows, overlaps, ranges crossing a 3-digit prefix). If loading still fails at runtime, it panics with the reason rather than silently returning `ErrNotFound`. |

**Performance** (13,107 ranges, 2-core Xeon 2.8 GHz, Go 1.24; source: `bench/results/baseline.txt`):

| Benchmark | Result |
|---|---|
| Lookup, random real pincodes | ~79 ns, 0 allocations |
| Lookup, same pincode every time (best case) | ~28 ns |
| Lookup in parallel, both cores | ~39 ns per lookup overall (~25M per second) |
| Invalid input rejected | ~41 ns |
| First-use load (parse and index) | ~15 ms, one time |
| Memory held after load | 1.9 MB |
| Size added to the binary | 540 KB |

Run `scripts/bench.sh` to measure. Run `scripts/bench.sh -compare bench/results/baseline.txt` to compare against the baseline with benchstat. Re-record the baseline when the data or the hardware changes.

**Alternatives considered**

| Option | Result | Decision |
|---|---|---|
| Database lookup | 0.5–5 ms per call | Rejected for the hot path. The SQL script is kept for analytics only. |
| Global binary search | ~100 ns | Replaced. |
| `map[prefix]` buckets | ~98 ns (hashing cancels the gain) | Rejected. |
| B-tree | No benefit for static data; worse cache locality | Rejected. |
| Prefix-array buckets | **~79 ns** | **Chosen.** |
| Direct 900k-slot table | ~2–5 ns, ~1.8 MB | Future option if lookups ever need more than ~10M per second per core. |

## 4. Data file

Columns: `pincode_from,pincode_till,city,state,iso3166_subdivision_code`. The current file has 13,107 non-overlapping ranges covering 6,532 cities, built from 19,300 pincodes.

Rules that the loader enforces:
- Ranges must not overlap.
- A range must not cross a 3-digit prefix (e.g. 560900–561100 must be split).
- Pincodes must be between 100000 and 999999.

**How the city was decided**

| Rule | Rows |
|---|---|
| Town: post office named after a town inside its pincode area, inside a municipal boundary, or at a town centre | 6,595 |
| Village: mapped to its sub-district HQ town (taluk / tehsil / mandal / block / circle) | 12,351 |
| Metro or state-capital rollup: neighbourhoods take the city name | 353 |
| No usable location: decided by name, nearest pincode or district HQ | 54 |

**Manual pincode overrides:** Kolkata 700001–700163, Jamshedpur 831001–831021, Hyderabad 500001–500115, Dharwad 580001–580011, Hubballi 580020–580032.

**Sources:** India Post all-India pincode directory (via data.gov.in, Dec 2021); data.gov.in pincode boundaries; LGD sub-district boundaries; Swachh Bharat Mission / West Bengal municipal boundaries (CC0, via ramSeraph/indian_admin_boundaries); GeoNames towns (CC BY 4.0, which requires attribution).

## 5. CSV caveats

1. **Data vintage.** The pincode list is from Dec 2021. Pincodes created later return `ErrNotFound`, or inherit a city if they fall inside an existing range.
2. **Unlisted pincodes inside a range.** Ranges merge neighbouring pincodes that share a city, so any unlisted number inside a range resolves to that city. Gaps between two different cities return `ErrNotFound`.
3. **One city per pincode.** A pincode that covers several villages or towns returns the city of its delivery post office.
4. **Village HQ by name.** For about 5,950 villages, no town matching the sub-district name was found, so the city is the sub-district's own name. This is usually the HQ, but it is wrong where a taluk isn't named after its HQ. Known cases are hand-mapped (Goa, Kerala, Pune's Haveli/Mulshi/Maval).
5. **Metro definition.** Delhi (all of NCT → "New Delhi"), Mumbai, Kolkata, Chennai, Bengaluru Urban and Hyderabad roll up by district or municipal area; other state capitals roll up by municipal boundary or radius. **Separate municipal corporations stay separate:** Thane, Navi Mumbai, Howrah, Noida, Ghaziabad, Gurugram, Pimpri-Chinchwad. Exception: municipalities inside an override range (e.g. Salt Lake, Dum Dum) take the override city.
6. **Source quality.** Post office coordinates were noisy (shared placeholder points, swapped lat/long, wrong district), so pincode boundaries were used instead (99% coverage). Some municipal boundaries were missing or oversized and were filtered out. GeoNames populations are census-2011 era.
7. **Spelling.** Current official names are used (Bengaluru, Mysuru, Kozhikode, Gurugram, Prayagraj). A few GeoNames or local variants may remain (e.g. "Sivagangai", "Nowrangapur").
8. **Duplicates.** One pincode (764063) had conflicting cities in the source; the majority value (Nowrangapur) was used.
9. **Verification.** The data was spot-checked (all metros, state capitals, Goa, the North-East, and random samples), not verified row by row. `city_mapping_method.csv`, kept outside the library, records how each pincode's city was decided.

## 6. Updating the data

1. Edit or regenerate `data/pincode_city_state_ranges.csv`. For a correction, add an override range and rebuild the ranges.
2. Run `go test ./...` to validate the file and the known lookups, then `scripts/bench.sh -compare bench/results/baseline.txt` to check speed.
3. Release a new library version. Consumers pick it up by bumping the dependency.
