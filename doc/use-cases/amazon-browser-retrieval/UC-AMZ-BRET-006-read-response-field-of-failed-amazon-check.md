# UC-AMZ-BRET-006 — Tell the cause from the returned page in a failed Amazon check's log line

**Primary actor:** Operator who installs and runs the server
**Supporting systems:** Repository installation documentation, PriceFollower application, server log
**Trigger:** An Amazon check failed and the operator reads its line in the server log next to the installation documentation's list of log lines.
**Origin:** Issue #210 (review comment R2-03 on #206): the documented failure line has no `response` field, so the start of the page Amazon returned may be read as part of the cause.
**Covers:** FR-AMZ-BRET-009, FR-AMZ-BRET-005

## Preconditions

- At least one Amazon check has failed and its line is in the server log (UC-AMZ-BRET-002).

## Main flow

1. The operator opens the section of the installation documentation that lists what the server log shows.
2. The documentation shows the failure line as `Amazon request failed url=… at=… status=… cause=… response=…`, with its fields in the order the server logs them.
3. One sentence says that `response` is the start of the page Amazon returned and that it can be empty.
4. The operator finds the failure line in the server log; its fields appear in the documented order, ending with `cause=… response=…`.
5. The operator reads the cause from `cause` only, and the text after `response=` as the start of the page Amazon returned, not as part of the cause.
6. The operator acts on the cause (UC-AMZ-BRET-002, step 4).

## Alternatives and errors

- **Amazon returned no page (for example the browser could not start):** the line ends with an empty `response=`; the documentation says this can happen, and the operator reads the cause from `cause` as in step 5.
- **Page without a detectable price:** the line `Amazon check found no price url=… at=… status=… cause=…` is documented as before; it is not a failed request.

## Postconditions

- The rest of the documented list of log lines, including the description of the causes and the line for a page without a detectable price, is unchanged.
- The application's behaviour and its log lines are unchanged.
