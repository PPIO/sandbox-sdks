#!/usr/bin/env python3
"""Confirm registry metadata and bytes before announcing a GitHub Release."""
import hashlib
import importlib.util
import pathlib
import sys
import time
import urllib.request
import urllib.error
spec = importlib.util.spec_from_file_location("release", pathlib.Path(__file__).with_name("release.py"))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


def verify(info, assets):
    component, version = info["component"], info["version"]
    headers = {"User-Agent": "sandbox-sdk-release-check"}
    def read(url):
        with urllib.request.urlopen(urllib.request.Request(url, headers=headers), timeout=30) as response:
            return response.read()
    base = "https://repo.maven.apache.org/maven2"
    files = [assets / f"ppio-sandbox-{version}{suffix}{signature}"
             for suffix in (".jar", "-sources.jar", "-javadoc.jar", ".pom")
             for signature in ("", ".asc")]
    for file in files:
        if not file.is_file():
            raise ValueError("Missing signed Java release artifact: " + file.name)
    for file in files:
        published = read(base + f"/com/ppio/ppio-sandbox/{version}/" + file.name)
        if hashlib.sha256(published).digest() != hashlib.sha256(file.read_bytes()).digest():
            raise ValueError("Maven artifact differs from the validated build")


def main():
    info = release.describe(sys.argv[1])
    if info["component"] == "go":
        raise ValueError("Go registry verification is not part of the release workflow")
    error_name = "unavailable"
    for attempt in range(24):
        try:
            verify(info, pathlib.Path("release-assets"))
            print(f"Verified registry artifacts for {info['package']} {info['version']}")
            return
        except (urllib.error.HTTPError, urllib.error.URLError, ValueError, KeyError, TimeoutError) as error:
            error_name = type(error).__name__
        if attempt < 23:
            time.sleep(5)
    raise SystemExit(f"Registry artifact verification failed ({error_name}); GitHub Release remains unannounced")


if __name__ == "__main__":
    main()
