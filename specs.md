# Table Header Change Verification Specification

## 1. Purpose

This feature compares the table headers of a previous-month report and a current-month report.

The system must:

- locate the intended table in each workbook;
- extract the complete header row;
- compare previous-month headers against current-month headers;
- notify the analyst when columns are added, removed, renamed, or reordered;
- retain a baseline and optional staging configuration for future runs;
- stop and require locator confirmation when the intended table cannot be identified reliably.

This is not a general-purpose spreadsheet understanding system. It is designed for recurring reports that are expected to retain at least some identifying structure between runs.

---

## 2. Core Concepts

### 2.1 File Mapping

Defines which previous-month file should be compared with which current-month file.

Example:

```text
previous_report.xlsx -> current_report.xlsx
```

### 2.2 Baseline Headers

The currently trusted header configuration used as a known reference for table identification and comparison state.

Example:

```text
[A, B]
```

### 2.3 Staging Headers

A changed header configuration detected in the current-month file.

Staging exists only when the current-month headers differ from the baseline.

Example:

```text
Baseline: [A, B]
Current:  [A, D, C]
Staging:  [A, D, C]
Diff:     +D, +C, -B
```

### 2.4 Locator Candidates

The known header configurations used to find the intended table.

Locator candidates may include:

- baseline headers;
- staging headers, when staging exists.

### 2.5 Previous-Month Source of Truth

At the start of the next run, the previous-month saved file is reread.

Its latest saved structure determines the effective baseline for the new comparison.

### 2.6 Normal Structural Drift

A header change where at least one known header survives and exactly one plausible table is found.

Example:

```text
Known: [A, B]
Found: [A, D, C]
```

### 2.7 Ambiguous Match

More than one plausible table matches the known locator candidates.

Example:

```text
Candidate 1: [A, D, C]
Candidate 2: [A, Z, K]
```

### 2.8 Complete Replacement

No known header from either baseline or staging survives.

Example:

```text
Known baseline: [A, B]
Known staging:  [A, D, C]
Found:          [E, F, G]
```

This requires confirmation that the new table is the intended replacement.

---

## 3. Assumptions

The implementation may rely on the following assumptions:

1. Each mapped file contains the same logical recurring report.
2. The intended table usually retains at least one known header between runs.
3. A single surviving known header is acceptable only when it identifies exactly one plausible table.
4. The complete header row can be extracted once the intended table is located.
5. Column order may be configured as significant or insignificant.
6. When no known header survives, the system must not guess.
7. When multiple plausible tables exist, the system must not choose silently.
8. The locator may require maintenance when the report changes beyond the supported assumptions.

---

## 4. High-Level Workflow

```text
Load file mapping
        |
        v
Load baseline and optional staging
        |
        v
Locate and reread previous-month table
        |
        v
Resolve effective baseline
        |
        v
Locate current-month table
        |
        v
Compare previous-month headers vs current-month headers
        |
        +--> Same
        |      - clear staging
        |      - keep effective baseline
        |      - no notification
        |
        +--> Different
               - notify analyst
               - keep effective baseline
               - save current headers as staging
```

---

## 5. State Resolution

### 5.1 No Existing Staging

Compare the current-month headers directly against the baseline.

#### Current matches baseline

```text
Baseline: [A, B]
Current:  [A, B]
```

Outcome:

- keep baseline;
- no staging;
- no notification.

#### Current differs from baseline

```text
Baseline: [A, B]
Current:  [A, D, C]
```

Outcome:

- keep baseline;
- save `[A, D, C]` as staging;
- notify analyst of `+D, +C, -B`.

---

### 5.2 Existing Staging

At the next run, reread the saved previous-month file.

Assume:

```text
Baseline: [A, B]
Staging:  [A, D, C]
```

#### Previous month now matches baseline

```text
Previous reread: [A, B]
```

Outcome:

- discard staging;
- keep baseline `[A, B]`.

#### Previous month matches staging

```text
Previous reread: [A, D, C]
```

Outcome:

- promote staging to baseline;
- clear staging;
- effective baseline becomes `[A, D, C]`.

#### Previous month matches neither baseline nor staging

```text
Previous reread: [A, Z, K]
```

Outcome:

- use the latest saved previous-month structure as the effective baseline;
- clear old staging;
- effective baseline becomes `[A, Z, K]`.

This is allowed only when the intended table was located confidently.

---

## 6. Table Identification Rules

The locator should evaluate baseline and staging as independent locator candidates.

Recommended search order:

```text
1. Exact staging match
2. Exact baseline match
3. Confident partial match using staging
4. Confident partial match using baseline
5. Ambiguous or no reliable match
```

### 6.1 Exact Match

All known locator headers are found in one candidate header row.

Outcome:

- accept automatically;
- extract the complete header row.

### 6.2 Confident Partial Match

At least one known header survives, and only one plausible table is found.

Example:

```text
Known: [A, B]
Found: [A, Z, K]
```

Outcome:

- accept automatically;
- extract `[A, Z, K]`;
- compare against the previous structure;
- record the match as partial.

### 6.3 Ambiguous Partial Match

At least one known header survives, but multiple candidate tables are plausible.

Example:

```text
Known: [A, B]

Candidate 1: [A, D, C]
Candidate 2: [A, Z, K]
```

Outcome:

- stop processing;
- return `AmbiguousMatch`;
- require locator confirmation or stronger configuration.

### 6.4 No Known Headers Survive

Example:

```text
Baseline: [A, B]
Staging:  [A, D, C]
Found:    [E, F, G]
```

Outcome:

- stop processing;
- return `LocatorConfirmationRequired`;
- require confirmation that `[E, F, G]` is the intended replacement;
- update the locator after confirmation.

---

## 7. Header Comparison

Once both tables are located, compare the complete extracted header rows.

The comparison result should contain:

```text
Added
Removed
Unchanged
OrderChanged
```

Example:

```text
Previous: [A, B]
Current:  [A, D, C]

Added:   [D, C]
Removed: [B]
```

Renamed columns are represented as one removal and one addition unless an explicit rename mapping exists.

---

## 8. Required Result Types

### 8.1 Table Location Status

```text
ExactMatch
PartialMatch
AmbiguousMatch
NotFound
ConfirmationRequired
```

### 8.2 Table Location Result

A table location result should contain:

```text
Status
SheetName
HeaderRow
ExtractedHeaders
MatchedHeaders
MissingKnownHeaders
AddedHeaders
MatchSource
CandidateMatches
```

`MatchSource` should indicate:

```text
Baseline
Staging
Both
```

### 8.3 Header Difference

```text
Added
Removed
Unchanged
OrderChanged
```

### 8.4 Verification State

```text
BaselineHeaders
StagingHeaders
StagedFileIdentity
```

Optional supporting metadata:

```text
BaselineSheet
BaselineHeaderRow
StagingSheet
StagingHeaderRow
```

### 8.5 Verification Outcome

```text
EffectiveBaseline
NewStaging
HeaderDifference
NotificationRequired
LocationStatus
FailureReason
```

---

## 9. Suggested Interfaces

The exact implementation language may differ, but responsibilities should remain separate.

### 9.1 Table Locator

```text
LocateTable(
    workbook,
    locatorCandidates
) -> TableLocationResult
```

Responsibility:

- scan workbook sheets and rows;
- evaluate candidate header rows;
- return exact, partial, ambiguous, or not-found results.

It must not update baseline or staging state.

### 9.2 Header Matcher

```text
MatchHeaders(
    knownHeaders,
    candidateHeaders
) -> HeaderMatch
```

Result:

```text
Matched
Missing
Added
MatchCount
MatchRatio
```

### 9.3 Header Comparator

```text
CompareHeaders(
    previousHeaders,
    currentHeaders
) -> HeaderDifference
```

Responsibility:

- compare two complete header sets;
- respect configured order sensitivity.

### 9.4 State Resolver

```text
ResolveVerificationState(
    existingState,
    previousMonthHeaders,
    currentMonthHeaders
) -> VerificationOutcome
```

Responsibility:

- resolve baseline and staging transitions;
- determine whether notification is required.

It must not read workbook files.

---

## 10. Failure Conditions

Processing must stop when:

1. no known baseline or staging header can be found;
2. multiple plausible tables match;
3. the workbook cannot be read;
4. the header row cannot be extracted;
5. the previous-month or current-month file is missing;
6. the located table is structurally invalid.

Suggested failure reasons:

```text
NoKnownHeadersMatched
MultiplePlausibleTablesFound
HeaderRowNotReadable
PreviousFileMissing
CurrentFileMissing
WorkbookReadFailure
```

---

## 11. Human Confirmation Boundary

Human confirmation is required only when the system cannot identify the intended table reliably.

Confirmation is not required for normal header changes.

### Automatic

```text
[A, B] -> [A, D, C]
```

Provided one plausible table is found.

### Confirmation required

```text
[A, B] / [A, D, C] -> [E, F, G]
```

Or:

```text
Candidate 1: [A, D, C]
Candidate 2: [A, Z, K]
```

The confirmation action should identify the correct table and update the locator configuration.

---

## 12. Non-Goals

This feature does not:

- understand arbitrary spreadsheet semantics;
- infer business meaning from unrelated headers;
- decide whether a structural change is correct;
- approve or reject analyst changes;
- guarantee table discovery after all known identifiers disappear;
- silently select between ambiguous candidates.

---

## 13. Acceptance Criteria

1. The system can locate a table using baseline headers.
2. The system can also use staging headers as locator candidates.
3. Exact matches are processed automatically.
4. A unique partial match with at least one known header is processed automatically.
5. Multiple partial matches return an ambiguity result.
6. Zero known-header matches require locator confirmation.
7. Complete extracted headers are compared, not only locator headers.
8. Added and removed columns are reported.
9. Differences create staging.
10. Matching current headers do not create staging.
11. Existing staging is resolved by rereading the previous-month saved file.
12. The latest confidently identified previous-month structure becomes the effective baseline.
13. The system does not silently guess when identification is ambiguous.
14. Locator confirmation and schema comparison remain separate concerns.
