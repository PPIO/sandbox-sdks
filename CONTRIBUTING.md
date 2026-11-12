# Contributing

Use a Go toolchain satisfying `sdk-go/go.mod`, JDK 11+ and Maven 3.9+. Python 3.9+ runs the release and independent
consumer checks. Start a feature or fix branch from main and open a pull request
with a description of the behavior change and its validation.

```sh
cd sdk-go
go build ./...
go vet ./...
go test -race ./...
cd ../sdk-java
mvn test
```

Run `python3 -m unittest discover -s .github/scripts/tests -v` and
`python3 release-smoke-tests/local_packages.py` from the root to validate release
infrastructure and installable packages. These checks use local fixtures, need
no API credentials and create no cloud resources. See
[consumer testing](release-smoke-tests/README.md) for isolated-cache and opt-in
live testing options.

Keep request names, response parsing, timeouts and errors aligned between Go and
Java. Migration APIs are not exposed by this SDK. Public examples
should begin with the PPIO facade. Add regression tests for behavior changes and
update CHANGELOG.md and relevant README sections.

Tests and examples must create their own resources and clean them up. Never use
hard-coded customer resource IDs, commit credentials, or print authentication
headers. Report security issues using [SECURITY.md](SECURITY.md).

Contributions are licensed under the [MIT License](LICENSE). Releases follow
[RELEASING.md](RELEASING.md); contributors do not need publishing credentials.
