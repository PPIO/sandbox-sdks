"""Exercise GitHub Release recovery without contacting GitHub."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "github-release.sh"


class GitHubReleaseRecovery(unittest.TestCase):
    def run_release(self, published, draft=False):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            assets = root / "release-assets"
            assets.mkdir()
            (assets / "source.tar.gz").write_bytes(b"validated source")
            (assets / "checksums.txt").write_bytes(b"validated checksums")
            remote = root / "published"
            remote.mkdir()
            (remote / "source.tar.gz").write_bytes(published)
            draft_file = root / "draft-state"
            draft_file.write_text("true" if draft else "false")
            binary = root / "bin"
            binary.mkdir()
            fake = binary / "gh"
            fake.write_text("#!" + sys.executable + "\n" + textwrap.dedent("""\
                import os, pathlib, shutil, sys
                args = sys.argv[1:]
                remote = pathlib.Path(os.environ['GH_ASSETS'])
                trace = pathlib.Path(os.environ['GH_TRACE'])
                draft_file = pathlib.Path(os.environ['GH_DRAFT_FILE'])
                action = args[1]
                with trace.open('a') as output:
                    output.write(action + '\\n')
                if action == 'view':
                    if '--json' in args:
                        if args[args.index('--json') + 1] == 'isDraft':
                            print(draft_file.read_text())
                        else:
                            print('\\n'.join(p.name for p in remote.iterdir()))
                elif action == 'download':
                    name = args[args.index('--pattern') + 1]
                    directory = pathlib.Path(args[args.index('--dir') + 1])
                    shutil.copy2(remote / name, directory / name)
                elif action == 'upload':
                    for name in args[3:]:
                        if name == '--repo':
                            break
                        source = pathlib.Path(name)
                        target = remote / source.name
                        if target.exists():
                            sys.exit(1)
                        shutil.copy2(source, target)
                elif action == 'edit':
                    if '--draft=false' not in args:
                        sys.exit(2)
                    draft_file.write_text('false')
                else:
                    sys.exit(2)
                """))
            fake.chmod(0o755)
            env = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"],
                       TAG="sdk-go/v0.0.1", CHANNEL="stable", GITHUB_REPOSITORY="PPIO/sandbox-sdks",
                       GH_ASSETS=str(remote), GH_TRACE=str(root / "trace"),
                       GH_DRAFT_FILE=str(draft_file))
            result = subprocess.run(["bash", str(SCRIPT)], cwd=root, env=env,
                                    capture_output=True, text=True)
            return result, (remote / "source.tar.gz").read_bytes(), \
                (remote / "checksums.txt").read_bytes() if (remote / "checksums.txt").exists() else None, \
                (root / "trace").read_text(), draft_file.read_text()

    def test_rerun_skips_identical_asset_and_uploads_missing_asset(self):
        result, source, checksums, trace, draft = self.run_release(b"validated source")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(source, b"validated source")
        self.assertEqual(checksums, b"validated checksums")
        self.assertEqual(trace.count("upload"), 1)
        self.assertEqual(draft, "false")

    def test_rerun_rejects_changed_published_asset_without_overwriting(self):
        result, source, checksums, trace, draft = self.run_release(b"different published source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(source, b"different published source")
        self.assertIsNone(checksums)
        self.assertNotIn("upload", trace)
        self.assertEqual(draft, "false")

    def test_rerun_publishes_existing_draft_after_validating_assets(self):
        result, source, checksums, trace, draft = self.run_release(b"validated source", draft=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(source, b"validated source")
        self.assertEqual(checksums, b"validated checksums")
        self.assertEqual(draft, "false")
        self.assertIn("edit", trace)


if __name__ == "__main__":
    unittest.main()
