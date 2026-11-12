#!/usr/bin/env bash
set -euo pipefail
output="${1:?output archive path required}"
epoch="${SOURCE_DATE_EPOCH:?SOURCE_DATE_EPOCH required}"
case "$epoch" in *[!0-9]*) echo "SOURCE_DATE_EPOCH must be an integer" >&2; exit 1 ;; esac
# An immutable tag must produce identical source bytes on every release attempt.
git archive --format=tar --mtime="@$epoch" HEAD:sdk-go | gzip -n > "$output"
