# Amazon searches frontend cleanups and back navigation

Status: todo (non-blocking PR review comments, triaged on 2026-10-07; not scheduled)

## Problem

- #91 `src/components/AmazonSearchesPage.tsx:130`: `addError` not cleared while the URL is edited.
- #91 `AmazonSearchesPage.tsx:46`, #92 `src/components/AmazonSearchItems.tsx:18,29`: `errorMessage`, formatters, rating labels and euro formatter duplicated or imported from another page.
- #92 `src/components/TrackedItemsPage.tsx:65`: effect not cancelled and refetches on every poll.
- #92 `src/App.tsx:310`: "Back to Amazon searches" goes to `/` (tab from in-memory state); an item opened from a search does not return to that search. No specification defines this navigation.

## Suggested work

Clear `addError` on change; move shared helpers to a utility module in a refactoring PR; cancel the effect and depend on stable inputs; specify, then implement, back navigation to the Amazon searches tab and to the originating search.

## Source

- PR reviews 5446175748 (#91), 5446176333 (#92)
