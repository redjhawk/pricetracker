# UC-RL-02 — Review the users list edge to edge below desktop width

**Primary actor:** Administrator
**Trigger:** The administrator opens the administrator page at a viewport narrower than 1056 px, where the users list is shown as stacked cards.
**Origin:** Issue #165, owner answers: lists must use the lateral margins and the empty part of each card; "all screen should use available space".
**Covers:** FR-RESPONSIVE-007, FR-RESPONSIVE-022, FR-RESPONSIVE-023, FR-RESPONSIVE-024

## Preconditions

- The administrator is logged in.
- At least one user exists.

## Main flow

1. The administrator opens the administrator page in a window, for example, 775 px wide.
2. The page title and the Add user action are at most 16 px from the screen edges and do not touch them.
3. Each user card spans the whole screen width, edge to edge. No page background shows to the left or right of the cards.
4. Inside each card, Username and Last login keep a small inner padding and are on the same line, each identified by its label or column name.
5. The administrator reads every username and last login without any sideways scrolling and uses Add user.

## Alternatives and errors

- **Phone width (320 px to 671 px):** the cards are still edge to edge. If Username and Last login do not fit side by side, Last login continues on the next line; both stay fully visible and identifiable.
- **Long username:** it wraps inside the card instead of widening the page.
- **Desktop (1056 px or more):** the users table with column headers keeps its current margins and layout.

## Postconditions

- The users list runs edge to edge below 1056 px, its content stays inset and uses the card width, and the page does not scroll sideways.
