# Commit step: issue #115 application information

## Gates before committing

- Specifications and contract consistent: yes (`app-information/functional.md`, `technical.md`, `API_SPECIFICATION.md` session `version`).
- Review findings REV-001..003: all fixed and rechecked, none critical ([decisions](decisions.md)); summary posted on issue #115.
- QA: API and unit checks pass; interface (Playwright) cases BLOCKED on the runner, Chromium lacks `libnspr4.so` ([qa](qa.md)).

## Progressive commits

At the user's request (issue #115, 2026-11-10), each stage was committed and pushed to `ai-dev/issue-115-20261110-0541` as it finished, so a cut-off run can resume. That branch is the work log; the PRs below repackage the same final content.

## Split (680+ changed lines > 500)

| Order | Branch | Base | Content | Changed lines |
|---|---|---|---|---|
| 1 | `ai-dev/issue-115-backend` | `ai-dev/issue-115-feature` | `internal/version`, session `version`, `build-release.sh`, specs, API, index, implementation record | 273 |
| 2 | `ai-dev/issue-115-frontend` | `ai-dev/issue-115-backend` | Information entry and modal, Playwright spec, review/decisions/QA/commit records | see PR |
| final | `ai-dev/issue-115-feature` | `master` | opened after both parts merge, `Closes #115` | exempt |

Checks run on part 1: `go vet ./...`, `go test ./...` pass. Part 2: `go test ./...`, `npm run build` pass.

## Limitation: `.github/workflows/release.yml`

The push of this change was refused (token without `workflow` scope). Without it CI releases report `dev` instead of the tag (FR-APP-INFO-007 not met in CI). A maintainer must apply:

```diff
   compilar:
     runs-on: arc-build
     steps:
+      - run: echo "TAG=v$(date -u +%Y%m%d%H%M%S)" >> "$GITHUB_ENV"
       - uses: actions/checkout@v7
@@
-      - run: ./scripts/build-release.sh amd64
+      - run: PRICEFOLLOWER_VERSION="$TAG" ./scripts/build-release.sh amd64
@@
-        run: gh release create "v$(date -u +%Y%m%d%H%M%S)" pricefollower-linux-amd64 pricefollower-linux-amd64.sha256 --target "$GITHUB_SHA" --notes "Compilación automática"
+        run: gh release create "$TAG" pricefollower-linux-amd64 pricefollower-linux-amd64.sha256 --target "$GITHUB_SHA" --notes "Compilación automática"
```
