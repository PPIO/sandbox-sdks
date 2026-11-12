#!/usr/bin/env bash
set -euo pipefail
: "${TAG:?}" "${CHANNEL:?}" "${GITHUB_REPOSITORY:?}"
assets=(release-assets/*)
[ -f "${assets[0]}" ] || { echo "No release assets to publish" >&2; exit 1; }
# Publish the Release only after the registry upload has succeeded.
if gh release view "$TAG" --repo "$GITHUB_REPOSITORY" >/dev/null 2>&1; then
  existing=$(gh release view "$TAG" --repo "$GITHUB_REPOSITORY" --json assets --jq '.assets[].name')
  downloaded=$(mktemp -d)
  trap 'rm -rf "$downloaded"' EXIT
  missing=()
  for asset in "${assets[@]}"; do
    name=${asset##*/}
    if printf '%s\n' "$existing" | grep -Fxq -- "$name"; then
      gh release download "$TAG" --pattern "$name" --dir "$downloaded" --repo "$GITHUB_REPOSITORY"
      if ! cmp -s "$asset" "$downloaded/$name"; then
        echo "Existing Release asset differs from validated build: $name" >&2
        exit 1
      fi
    else
      missing+=("$asset")
    fi
  done
  if [ "${#missing[@]}" -gt 0 ]; then
    gh release upload "$TAG" "${missing[@]}" --repo "$GITHUB_REPOSITORY"
  fi
  draft=$(gh release view "$TAG" --repo "$GITHUB_REPOSITORY" --json isDraft --jq '.isDraft')
  case "$draft" in
    true) gh release edit "$TAG" --draft=false --repo "$GITHUB_REPOSITORY" ;;
    false) ;;
    *) echo "Could not determine GitHub Release draft state" >&2; exit 1 ;;
  esac
else
  args=(release create "$TAG" --verify-tag --title "$TAG" --generate-notes --repo "$GITHUB_REPOSITORY")
  if [ "$CHANNEL" != stable ]; then args+=(--prerelease --latest=false); fi
  gh "${args[@]}" "${assets[@]}"
fi
