#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel)
cd "$repository_root"

if [[ -n $(git status --porcelain) ]]; then
  echo "release: the working tree must be clean" >&2
  exit 1
fi

branch=$(git branch --show-current)
if [[ "$branch" != "main" ]]; then
  echo "release: switch to main before releasing (current: $branch)" >&2
  exit 1
fi

# Fetch the branch without importing tags: a failed release may have left a
# local tag that conflicts with an already published remote tag.
if ! git fetch --no-tags origin refs/heads/main:refs/remotes/origin/main; then
  echo "release: could not refresh origin/main" >&2
  exit 1
fi
if ! git merge-base --is-ancestor refs/remotes/origin/main HEAD; then
  echo "release: main is behind or has diverged from origin/main; integrate the remote commits before releasing" >&2
  exit 1
fi
if ! remote_tags=$(git ls-remote --tags --refs origin 'refs/tags/v*'); then
  echo "release: could not read release tags from origin" >&2
  exit 1
fi

# The nearest tag can be older than another published release. Find the highest
# stable version across local and remote tags, without replacing local refs.
tag_candidates=$(git tag --list 'v[0-9]*')
while read -r object ref; do
  if [[ -n "$ref" ]]; then
    tag_candidates+=$'\n'"${ref#refs/tags/}"
  fi
done <<< "$remote_tags"

highest_major=-1
highest_minor=-1
highest_patch=-1
while IFS= read -r candidate; do
  if [[ ! "$candidate" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    continue
  fi
  major=$((10#${BASH_REMATCH[1]}))
  minor=$((10#${BASH_REMATCH[2]}))
  patch=$((10#${BASH_REMATCH[3]}))
  if (( major > highest_major ||
        (major == highest_major && minor > highest_minor) ||
        (major == highest_major && minor == highest_minor && patch > highest_patch) )); then
    highest_major=$major
    highest_minor=$minor
    highest_patch=$patch
  fi
done <<< "$tag_candidates"

suggested_version="v0.1.0"
if (( highest_major >= 0 )); then
  suggested_version="v${highest_major}.${highest_minor}.$((highest_patch + 1))"
fi

read -r -p "Release version [$suggested_version]: " release_version
release_version=${release_version:-$suggested_version}
if [[ "$release_version" != v* ]]; then
  release_version="v$release_version"
fi
if [[ ! "$release_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "release: version must use semantic versioning, for example v0.2.3" >&2
  exit 1
fi
if git rev-parse --quiet --verify "refs/tags/$release_version" >/dev/null; then
  echo "release: tag $release_version already exists" >&2
  exit 1
fi
if ! remote_tag=$(git ls-remote --tags origin "refs/tags/$release_version"); then
  echo "release: could not check whether $release_version exists on origin" >&2
  exit 1
fi
if [[ -n "$remote_tag" ]]; then
  echo "release: tag $release_version already exists on origin" >&2
  exit 1
fi

commit=$(git rev-parse --short HEAD)
read -r -p "Tag $commit as $release_version and push main plus the tag? [y/N] " confirmation
if [[ "$confirmation" != "y" && "$confirmation" != "Y" ]]; then
  echo "Release cancelled."
  exit 0
fi

make check
make cross-build VERSION="$release_version"

if [[ -n $(git status --porcelain) ]]; then
  echo "release: verification changed the working tree; refusing to tag" >&2
  exit 1
fi

git tag --annotate "$release_version" --message "Release $release_version"
if ! git push --atomic origin main "$release_version"; then
  git tag --delete "$release_version"
  echo "release: push failed; deleted local tag $release_version" >&2
  exit 1
fi

echo "Released $release_version from $commit."
