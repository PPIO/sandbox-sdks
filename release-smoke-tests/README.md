# Local Go and Java package consumers

These independent projects verify that customers can install and use the built SDK packages. By default they call local mock services, require no API key, create no cloud sandboxes and publish no packages or tags.

## Build and verify both languages

Requirements: Python 3.9+, a Go toolchain satisfying `sdk-go/go.mod`, JDK 11+ and Maven 3.9+. Run from the repository root:

```bash
CONSUMER_ROOT="$(mktemp -d /tmp/ppio-consumer.XXXXXX)"
python3 release-smoke-tests/local_packages.py --output "$CONSUMER_ROOT"
```

Use `--language go` or `--language java` to select one language. The output directory must be empty and outside the repository. Artifacts and reports are retained after failures for diagnosis.

Java downloads public dependencies from Maven Central on the first run. To use an existing Maven cache in an isolated repository and run offline:

```bash
python3 release-smoke-tests/local_packages.py \
  --language java \
  --maven-cache "$HOME/.m2/repository" \
  --offline
```

The script creates empty Maven settings and does not read personal registry credentials. It removes previously cached PPIO packages from the cache copy, so this run must build and install the SDK. Your original Maven cache remains intact.

## Installation checks

| Language | SDK artifact | Consumer dependency | Source isolation |
| --- | --- | --- | --- |
| Go | Standard module-proxy `.mod`, `.info` and `.zip` files | Independent `go.mod` requiring `github.com/PPIO/sandbox-sdks/sdk-go` | Disables go.work, rejects replace directives, uses a fresh GOMODCACHE and records `go list -m -json` provenance |
| Java | Maven-built JAR and POM | Independent `pom.xml` with `com.ppio:ppio-sandbox` | Builds the SDK and consumer separately; asserts classes come from a JAR and verifies its embedded version |

Packages use the version declared by the current branch. The script reads main, alpha and beta metadata automatically; tests require no tags. Go modules distribute source archives that customers compile in their own projects, so this test verifies that installation path.

The output contains:

- `result.json`: versions, consumer paths and the isolated Maven repository.
- `go-proxy/`, `go-modules/`, `go-consumer/`: module artifacts, cache and independent project.
- `go-package-origin.json`: the Go SDK provenance, with no replace directive.
- `m2/`, `java-sdk-build/`, `java-consumer/`: isolated Maven repository, SDK build copy and consumer.
- `java-consumer/target/surefire-reports/`: consumer test results. The console also prints the loaded JAR path.

Consumers create and connect sandboxes through the high-level `PPIO` facade, with connection options on the client. Go uses `client.CodeInterpreter`. Java uses `PPIO.createSandbox/connectSandbox`, then `CodeInterpreterSandbox.wrap` for interpreter support. Public subpackages remain available for option and result types; separate cases verify lower-level compatibility.

Tests use public APIs without reflection or package-private access. They cover binary file streams, missing files, command success/errors/timeouts, interpreter authentication, NDJSON output and callbacks, context options, reconnection, desktop connection, and Java directory watching, template serialization and build errors.

The full SDK suites additionally cover template archives, registry image inheritance, Git, PTY, directory watching, desktop process management and option combinations. Consumers focus on the installed public API rather than duplicating every internal test.

## Live acceptance using the same packages

After local package validation, reuse the output directory for explicit live acceptance:

```bash
# Set PPIO_API_KEY securely in your terminal. Do not commit credentials or .env files.
python3 release-smoke-tests/local_packages.py --run-live "$CONSUMER_ROOT" --language go
python3 release-smoke-tests/local_packages.py --run-live "$CONSUMER_ROOT" --language java
```

`--run-live` is opt-in. Each language creates one real sandbox and incurs resource usage. Tests destroy it in finally/cleanup and print its ID for recovery if cleanup fails. The default sandbox lifetime is 300 seconds.

| Variable | Purpose |
| --- | --- |
| `PPIO_API_KEY` | Required API key |
| `PPIO_DOMAIN` | Cluster domain; defaults to the SDK's default cluster |
| `PPIO_API_URL` | Control-plane override for a specific or self-hosted environment |
| `PPIO_CODE_TEMPLATE` | Python-capable interpreter template; defaults to `code-interpreter-v1` |

Live acceptance creates a sandbox, streams a file, verifies its contents with a command, runs Python and checks `42`, creates a context and variable, reconnects through the installed package, verifies the variable survived, deletes the context and destroys the sandbox.

These assertions read actual service output. Without a real API key and suitable template, only local packaging and protocol validation can be claimed.

Desktop, template-build and volume/Secret acceptance need the corresponding templates, images and permissions. After the common flow, validate:

1. Templates: relative COPY and symlinks, inherited PATH/WORKDIR/start commands, and logs/steps for failed builds.
2. Desktop: create a sandbox, launch an app, capture a screenshot, send input, connect VNC to the same display, then destroy it.
3. Data resources: create temporary volumes/Secrets, use and delete them, and check permissions and legacy-cluster restrictions.

## Test published packages

Reuse the consumer code after publication, changing only the dependency source.

For Go, use the normal proxy, an isolated cache and the selected version, with no replace directive:

```bash
GOWORK=off go get github.com/PPIO/sandbox-sdks/sdk-go@v0.0.1
GOWORK=off go test -count=1 -v ./...
```

For Java, use normal Maven settings and select the target version:

```bash
mvn -Dppio.version=2.1.2 test
```

These versions are examples; select the release under test. Java stable and prerelease packages use Maven Central without GitHub Packages settings or credentials. Public registry availability needs separate verification; local proxy/JAR installation does not establish remote downloadability.
