# No red phase for issue #251

The change is itself test code: the fix (the stub written by `writeModule` declares `const DownloadMarker = ""`) lives in `internal/amazon/amazontest/server_dependencies_test.go`, the file the TDD red commit locks. If this stage only added the build check, the developer could not then change the stub, so the red-then-green sequence cannot run in the factory. As the technical specification decides, this stage delivers the whole change; the development stage changes no code and only sets the statuses of FR-TEST-HYG-014–017 and TS-TEST-HYG-016–020 to implemented.

What this stage changed, in `server_dependencies_test.go` only:

- the stub declares `DownloadMarker`;
- the TC-003 case table moved to the package-level `fixtureCases` (type `fixtureCase`), with the same entries, subtest names and `fails` values;
- new `buildModule` (offline `go build ./...` with `fixtureEnv`) and `TestFixtureModulesBuild_Issue251_TC001`;
- the header comment lists where each issue #251 case is proved.

The red state is proved inside the new test: its control subtest (the stub as on `master`, with `helperMain`) must fail `go build ./...` naming `DownloadMarker`. Checked by hand as well: with the stub reverted to `package amazontest\n`, rows a, c and the renamed-import corner of the build test fail with `undefined: amazontest.DownloadMarker` / `undefined: helper.DownloadMarker`, and nothing else fails.

Where each test case is proved:

- **TC-001**: `TestFixtureModulesBuild_Issue251_TC001`. Every valid entry of `fixtureCases` and the `plainMain` module build offline; rows f and g and the control module do not. It runs in every `go test ./...` with no flag.
- **TC-002**: `TestListServerDependencies_TC003_FixtureModules`, rows a–e and the three corners over `fixtureCases`, with the same subtest names, files and expected results.
- **TC-003**: rows f, g and h of the same test, unchanged, run with the new stub.
- **TC-004**: repository check on the diff (outside `doc/`, only `internal/amazon/amazontest/server_dependencies_test.go`) and the gates. `go vet ./...` and `go test -count=1 ./...` pass.
