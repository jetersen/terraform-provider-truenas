#!/usr/bin/env bash
# Publish the clean snapshot to the PUBLIC repo.
#
# Builds a single-commit orphan from main's tree with the internal lab runbook
# and the publish tooling excluded, and TESTING.md scrubbed of lab topology,
# then (with --push) force-pushes it to the public repo's main. No history is
# published. main and the private mirror are untouched.
#
#   scripts/publish-public.sh          # DRY RUN: build + verify, do not push
#   scripts/publish-public.sh --push   # also force-push to the public repo
#
# Config (env overrides):
#   PUBLIC_REMOTE  git remote for the public repo (default: publish)
#   SOURCE_BRANCH  branch to snapshot (default: main)
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

PUBLIC_REMOTE="${PUBLIC_REMOTE:-publish}"
SOURCE_BRANCH="${SOURCE_BRANCH:-main}"
TMP_BRANCH="__publish_snapshot"
DO_PUSH=0
[ "${1:-}" = "--push" ] && DO_PUSH=1

# Files kept in the private repo but NEVER published: the internal lab runbook,
# and the publish tooling (which names the lab tokens it scrubs).
EXCLUDE=(
  MANUAL-TESTS.md
  scripts/publish-public.sh
  scripts/scrub-public-testing.py
  docs-dev/registry-publish-runbook.md
)

# --- guards ---------------------------------------------------------------
if [ -n "$(git status --porcelain)" ]; then
  echo "ABORT: working tree not clean — commit/stash first:" >&2
  git status --short >&2
  exit 1
fi
git rev-parse --verify --quiet "$SOURCE_BRANCH" >/dev/null || {
  echo "ABORT: no branch '$SOURCE_BRANCH'." >&2; exit 1; }
if [ "$DO_PUSH" -eq 1 ] && ! git remote get-url "$PUBLIC_REMOTE" >/dev/null 2>&1; then
  echo "ABORT: no remote '$PUBLIC_REMOTE'. Add it or set PUBLIC_REMOTE." >&2
  exit 1
fi

START_BRANCH="$(git branch --show-current)"
cleanup() { git checkout -q "$START_BRANCH" 2>/dev/null || git checkout -q "$SOURCE_BRANCH" 2>/dev/null || true
            git branch -D "$TMP_BRANCH" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# --- build the orphan snapshot from SOURCE_BRANCH's tree ------------------
git branch -D "$TMP_BRANCH" >/dev/null 2>&1 || true
git checkout -q --orphan "$TMP_BRANCH" "$SOURCE_BRANCH"

for f in "${EXCLUDE[@]}"; do
  git rm --cached --quiet "$f" 2>/dev/null || true
done
python3 scripts/scrub-public-testing.py
git add TESTING.md
git commit -q -m "Initial commit"

# --- verify ---------------------------------------------------------------
echo "== snapshot verification =="
echo "  commits (want 1): $(git rev-list --count HEAD)"
echo "  author: $(git log -1 --format='%an <%ae>')"
fail=0
for f in "${EXCLUDE[@]}"; do
  if git ls-files --error-unmatch "$f" >/dev/null 2>&1; then
    echo "  EXCLUDED FILE STILL PRESENT: $f"; fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "  excluded files absent: ok"
labhits=$(git grep -cE "192\.168\.1\.|10\.220\.16\.188|\bpve\b|plan20-ha|tftest|tfipa" -- TESTING.md || true)
echo "  TESTING.md lab tokens (want 0): ${labhits:-0}"; [ "${labhits:-0}" != "0" ] && fail=1
if git grep -q wmoh -- . ; then echo "  wmoh FOUND (bad)"; fail=1; else echo "  wmoh: none"; fi
echo "  files changed vs $SOURCE_BRANCH (expect the excludes + TESTING.md):"
git diff "$SOURCE_BRANCH" "$TMP_BRANCH" --stat | sed 's/^/    /'

if [ "$fail" -ne 0 ]; then
  echo "RESULT: verification FAILED — not pushing." >&2
  exit 1
fi

# --- push (only with --push) ---------------------------------------------
if [ "$DO_PUSH" -eq 1 ]; then
  echo "== force-pushing snapshot -> $PUBLIC_REMOTE main =="
  git push --force "$PUBLIC_REMOTE" "$TMP_BRANCH:main"
  echo "Published. Public repo main is now the clean snapshot ($(git rev-parse --short HEAD))."
  echo "NOTE: any pre-existing dependabot branches / closed-PR refs on the public"
  echo "repo still descend from earlier pushes; clear them via branch delete +"
  echo "a GitHub Support gc if this repo ever carried full history."
else
  echo "== DRY RUN: snapshot built and verified, NOT pushed. Re-run with --push to publish. =="
fi
