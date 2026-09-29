#!/usr/bin/env bash
# Publish the clean snapshot to the PUBLIC repo.
#
# Builds a snapshot of main's tree with the internal lab runbook and the publish
# tooling excluded, and TESTING.md scrubbed of lab topology, then (with --push)
# commits it ON TOP of the public repo's current main and pushes without force.
#
# History is linear: each release adds one commit to the public repo, so open
# pull requests keep a valid merge base and are not auto-closed by a rewrite.
# The published tree is copied, not the private history — no private commits are
# ever exposed. main and the private mirror are untouched.
#
# On the very first publish (public main does not yet exist) the snapshot is an
# orphan root commit; every publish after that builds on the previous one.
#
#   scripts/publish-public.sh          # DRY RUN: build + verify, do not push
#   scripts/publish-public.sh --push   # also push to the public repo (no force)
#
# Config (env overrides):
#   PUBLIC_REMOTE  git remote for the public repo (default: publish)
#   SOURCE_BRANCH  branch to snapshot (default: main)
#   PUBLISH_MSG    commit message (default: "Release <latest tag on SOURCE_BRANCH>")
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
  docs-dev/api-coverage-gaps.md
)

# --- guards ---------------------------------------------------------------
if [ -n "$(git status --porcelain)" ]; then
  echo "ABORT: working tree not clean — commit/stash first:" >&2
  git status --short >&2
  exit 1
fi
git rev-parse --verify --quiet "$SOURCE_BRANCH" >/dev/null || {
  echo "ABORT: no branch '$SOURCE_BRANCH'." >&2; exit 1; }
if ! git remote get-url "$PUBLIC_REMOTE" >/dev/null 2>&1; then
  if [ "$DO_PUSH" -eq 1 ]; then
    echo "ABORT: no remote '$PUBLIC_REMOTE'. Add it or set PUBLIC_REMOTE." >&2
    exit 1
  fi
  echo "WARN: no remote '$PUBLIC_REMOTE'; dry run treats this as first publish." >&2
fi

# Commit message: the release tag on SOURCE_BRANCH, else a generic label.
if [ -z "${PUBLISH_MSG:-}" ]; then
  if TAG=$(git describe --tags --abbrev=0 "$SOURCE_BRANCH" 2>/dev/null); then
    PUBLISH_MSG="Release $TAG"
  else
    PUBLISH_MSG="Publish snapshot"
  fi
fi

# --- locate the public base -----------------------------------------------
# Fetch the public main so we can build on top of it. If it does not exist
# yet, this is the first publish and we start an orphan root commit.
BASE_REF=""
if git remote get-url "$PUBLIC_REMOTE" >/dev/null 2>&1; then
  if git fetch --quiet "$PUBLIC_REMOTE" main 2>/dev/null; then
    BASE_REF="$PUBLIC_REMOTE/main"
  fi
fi

START_BRANCH="$(git branch --show-current)"
# Force the checkout back: building the snapshot leaves the excluded files in
# the working tree as untracked, which would make a plain `git checkout` refuse
# to switch back ("untracked files would be overwritten"). -f restores them
# from the target branch (same content) so cleanup always lands on the start.
cleanup() { git checkout -qf "$START_BRANCH" 2>/dev/null || git checkout -qf "$SOURCE_BRANCH" 2>/dev/null || true
            git branch -D "$TMP_BRANCH" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# --- build the snapshot from SOURCE_BRANCH's tree -------------------------
git branch -D "$TMP_BRANCH" >/dev/null 2>&1 || true
if [ -n "$BASE_REF" ]; then
  # Parent = current public main (ancestry preserved). Overlay SOURCE_BRANCH's
  # tree exactly: read-tree -u resets index+worktree to the source tree while
  # HEAD stays at the public commit, so the new commit's parent is public main
  # and its tree equals scrubbed SOURCE_BRANCH.
  echo "== building on public base $BASE_REF ($(git rev-parse --short "$BASE_REF")) =="
  git checkout -q -B "$TMP_BRANCH" "$BASE_REF"
  git read-tree -u --reset "$SOURCE_BRANCH"
else
  # First publish: no public history to build on — orphan root commit.
  echo "== first publish: no $PUBLIC_REMOTE/main, starting orphan root =="
  git checkout -q --orphan "$TMP_BRANCH" "$SOURCE_BRANCH"
fi

for f in "${EXCLUDE[@]}"; do
  git rm --cached --quiet "$f" 2>/dev/null || true
done
python3 scripts/scrub-public-testing.py
git add TESTING.md

# Nothing changed since the last publish? Skip the empty commit and the push.
if [ -n "$BASE_REF" ] && git diff --cached --quiet "$BASE_REF" -- ; then
  echo "== no changes vs $BASE_REF — nothing to publish. =="
  exit 0
fi

git commit -q -m "$PUBLISH_MSG"

# --- verify ---------------------------------------------------------------
echo "== snapshot verification =="
echo "  message: $PUBLISH_MSG"
echo "  public history depth after this commit: $(git rev-list --count HEAD)"
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
  echo "== pushing snapshot -> $PUBLIC_REMOTE main (fast-forward, no force) =="
  # No --force: if public main advanced (a commit landed directly on it), the
  # push is rejected rather than silently clobbering it. Fetch + re-run then.
  if ! git push "$PUBLIC_REMOTE" "$TMP_BRANCH:main"; then
    echo "PUSH REJECTED: $PUBLIC_REMOTE/main moved since fetch. Re-run to rebuild" >&2
    echo "on the new base, or reconcile the direct commit first." >&2
    exit 1
  fi
  echo "Published. $PUBLIC_REMOTE/main is now $(git rev-parse --short HEAD) on linear history."
else
  echo "== DRY RUN: snapshot built and verified, NOT pushed. Re-run with --push to publish. =="
fi
