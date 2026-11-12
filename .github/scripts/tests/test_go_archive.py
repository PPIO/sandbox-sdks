"""The Go source asset must be byte-identical on an immutable-tag rerun."""
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "go-archive.sh"


class GoArchive(unittest.TestCase):
    def test_same_commit_produces_same_archive_after_checkout_times_change(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def git(*args):
                subprocess.run(["git", *args], cwd=root, check=True, capture_output=True)
            git("init", "-q", "-b", "main")
            git("config", "user.name", "Archive Test")
            git("config", "user.email", "archive@example.com")
            source = root / "sdk-go/go.mod"
            source.parent.mkdir()
            source.write_text("module github.com/PPIO/sandbox-sdks/sdk-go\n")
            git("add", ".")
            git("commit", "-qm", "source")
            env = dict(os.environ, SOURCE_DATE_EPOCH="1700000000")
            first, second = root / "first.tar.gz", root / "second.tar.gz"
            for output in (first, second):
                result = subprocess.run(["bash", str(SCRIPT), str(output)], cwd=root,
                                        env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                os.utime(source, (1700000001, 1700000001))
            self.assertEqual(first.read_bytes(), second.read_bytes())
            self.assertEqual(first.read_bytes()[4:8], b"\0\0\0\0")
            with tarfile.open(first, "r:gz") as archive:
                member = archive.getmember("go.mod")
                self.assertEqual(member.mtime, 1700000000)
                self.assertEqual(archive.extractfile(member).read(),
                                 b"module github.com/PPIO/sandbox-sdks/sdk-go\n")


if __name__ == "__main__":
    unittest.main()
