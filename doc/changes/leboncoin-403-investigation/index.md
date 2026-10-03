# LeBoncoin 403 investigation

Date: 2026-10-03. Status: investigation independently verified; ready for documentation commit. Git's successful output and the coordinator's final handoff establish the commit result.

User request: test HTTP/browser behavior for [listing 3245888872](https://www.leboncoin.fr/ad/voitures/3245888872) until a working acquisition method is identified. The user reports normal browser access on the same network and completed the human verification in the isolated test browser when requested.

This is a diagnostic investigation, not an implemented product change. New functional/technical feature specifications, API changes and product-interface QA are not applicable. The scope and success criterion were the user's supplied URL and successful retrieval of its real listing data; no new application behavior was invented. Any proposed cookie configuration or browser integration must enter the normal specification-led implementation workflow separately.

- [Experiments, technical findings and API assessment](report.md).
- [Independent review](review.md).
- [Independent decisions](decisions.md).
- [Independent verification](qa.md).
- [Documentation commit](commit-step.md).

Roles: coordinator ran browser/HTTP experiments; `/root/leboncoin_diagnostic_assessment` independently assessed the existing architecture; `/root/leboncoin_probe_developer` authored a temporary harness invoking the actual collector. Review, adjudication and verification use separate agents. No production source or dependency was changed; all probes, profiles and cookie values remain outside the committed documentation.

Review: `/root/leboncoin_investigation_reviewer` identified one factual wording error about extraction before a screenshot timeout. `/root/leboncoin_investigation_adjudicator` classified it noncritical and required correction. The coordinator corrected it; the reviewer independently verified the correction with no new findings. `/root/leboncoin_investigation_qa` owns the final live cookie/no-cookie comparison.

Independent QA passed: one fresh Go request without the cookie returned 403; one with the validated cookie and current headers returned 200 and the expected title/price. Report revision, ARMv6 build metadata and unchanged production files were verified. No findings remain open; no product implementation or long-term reliability is implied.
