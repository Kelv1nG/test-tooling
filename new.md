# Previous-Month Table Lookup Scenarios

## Scenario Matrix

| ID | Scenario | Example | Result |
|---|---|---|---|
| A | Previous month matches staging | Baseline `[A, B]`, Staging `[A, D, C]`, Previous `[A, D, C]` | `FoundExact` via staging |
| B | Previous month matches baseline | Baseline `[A, B]`, Previous `[A, B]` | `FoundExact` via baseline |
| C | Previous contains the full locator plus extra columns | Baseline `[A, B]`, Previous `[A, B, Q, K]` | `FoundExact`; extract full block `[A, B, Q, K]` |
| D | Partial baseline match only | Baseline `[A, B]`, Previous `[A, Z, K]` | `FoundPartial` only when exactly one plausible table exists |
| E | Partial staging match only | Staging `[A, D, C]`, Previous `[D, Z, K]` | `FoundPartial` only when exactly one plausible table exists |
| F | Partial match against both baseline and staging | Baseline `[A, B]`, Staging `[A, D, C]`, Previous `[A, B, D, Z]` | `FoundPartial` via both, provided both point to the same contiguous block |
| G | Exact match for one and partial match for the other | Baseline `[A, B]`, Staging `[A, D, C]`, Previous `[A, B, D]` | `FoundExact`; exact match wins, but extract the complete block |
| H | Multiple tables match baseline | Table 1 `[A, B, Q]`, Table 2 `[A, B, K]` | `Ambiguous` |
| I | Multiple tables match staging | Table 1 `[A, D, C]`, Table 2 `[A, D, C, X]` | `Ambiguous` |
| J | Baseline and staging match different tables | Table 1 matches baseline, Table 2 matches staging | `Ambiguous` |
| K | One surviving header appears in multiple tables | Table 1 `[A, Z, K]`, Table 2 `[A, X, Y]` | `Ambiguous` |
| L | Completely different structure | Baseline `[A, B]`, Staging `[A, D, C]`, Previous `[E, F, G]` | `NotFound`; locator confirmation required |
| M | Known headers are split across different blocks | Block 1 `[A, Q]`, Block 2 `[B, K]` | `NotFound`; do not combine headers from separate blocks |
| N | No stable intersection and neither full locator matches | Baseline `[A, B]`, Staging `[C, D]`, Previous `[A, C, X]` | Prefer `NotFound` or confirmation required for the initial version |
| O | Header boundary is unclear | `[A, B, blank, C, D]` | Extraction error unless blank headers are explicitly supported |
| P | Duplicate headers | `[A, B, B, C]` | Validation or extraction error unless duplicate headers are allowed |
| Q | Workbook cannot be read | Corrupted, password-protected, inaccessible, or invalid workbook | File-processing error |
| R | Previous-month file is missing | Expected previous file is unavailable | Verification-gap error |

## Main Lookup Statuses

| Status | Meaning |
|---|---|
| `FoundExact` | Exactly one table fully contains the baseline or staging locator headers |
| `FoundPartial` | No exact match exists, but exactly one plausible table contains surviving known headers |
| `Ambiguous` | More than one table is plausible |
| `NotFound` | No table can be identified reliably |

## Suggested Lookup Result

```text
Status
MatchedBy          // baseline, staging, or both
MatchedHeaders
ExtractedHeaders
CandidateTables
FailureReason
```

## Resolution Rules

```text
One exact candidate
â†’ FoundExact

No exact candidate, but one unique partial candidate
â†’ FoundPartial

More than one plausible candidate
â†’ Ambiguous

No plausible candidate
â†’ NotFound
```

## Important Rules

1. Baseline and staging are searched as separate locator configurations.
2. A locator is a required subset of the complete table headers.
3. When a locator matches, return the complete contiguous header block.
4. Do not generate and search every possible subset of baseline and staging.
5. Do not combine matching headers found in separate table blocks.
6. Exact matches take priority over partial matches.
7. When baseline and staging point to different tables, return `Ambiguous`.
8. When no known headers survive, require locator confirmation rather than guessing.
9. Workbook access failures and missing files are operational errors, not table lookup results.