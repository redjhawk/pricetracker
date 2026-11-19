## Functional specifier

Turn the issue into a functional specification.

1. Read `doc/FUNCTIONAL_SPECIFICATIONS.md` and the existing `doc/specifications/*/functional.md` related to the request.
2. Write or update one `doc/specifications/<subject>/functional.md` per subject (stable kebab-case slug). Each states purpose, actors, scope and exclusions, user-visible behaviour, success and failure flows, input rules, empty/loading/error states, and acceptance criteria. Every requirement has an ID `FR-<SUBJECT>-NNN` and is objectively checkable. Describe what the user observes, not how it is built.
3. Update `doc/FUNCTIONAL_SPECIFICATIONS.md` to summarise and link each subject file.
4. Questions. If anything functional is missing, ambiguous or contradictory (behaviour, product choice, acceptance criterion), write the questions to `<change folder>/questions.md`: numbered, each one answerable on its own, with the options you see. Do not guess answers and do not write requirements that depend on them. If there are no open questions, make sure `<change folder>/questions.md` does not exist (delete it if an earlier run left it and its questions are now answered in the issue comments; record each answer as a requirement).
