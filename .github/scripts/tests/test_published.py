import importlib.util
import pathlib
import sys
import unittest
from unittest.mock import patch

SCRIPTS = pathlib.Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('verify_published', SCRIPTS / 'verify-published.py')
published = importlib.util.module_from_spec(spec)
spec.loader.exec_module(published)


class PublishedPackageGate(unittest.TestCase):
    def test_go_tag_cannot_enter_java_registry_verification(self):
        with patch.object(sys, 'argv', ['verify-published.py', 'sdk-go/v0.0.1']):
            with self.assertRaisesRegex(ValueError, 'Go registry verification is not part of the release workflow'):
                published.main()


if __name__ == '__main__':
    unittest.main()
