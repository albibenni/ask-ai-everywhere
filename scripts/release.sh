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

current_tag=$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)
suggested_version="v0.1.0"
if [[ "$current_tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  suggested_version="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))"
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
git push --atomic origin main "$release_version"

echo "Released $release_version from $commit."
