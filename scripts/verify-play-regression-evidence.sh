#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
fix_commit="${1:-e67bdaee72df2a2dec93600820777072095c0a4c}"
buggy_commit="${fix_commit}^"
current_head="$(git rev-parse HEAD)"

if ! git merge-base --is-ancestor "$fix_commit" "$current_head"; then
  printf 'Current HEAD (%s) does not contain play fix commit %s\n' "$current_head" "$fix_commit" >&2
  exit 1
fi

temp_dir="$(mktemp -d)"
head_tree="$temp_dir/head"
buggy_tree="$temp_dir/buggy"
temp_test_rel="internal/cli/play_regression_evidence_test.go"

cleanup() {
  git -C "$repo_root" worktree remove --force "$head_tree" >/dev/null 2>&1 || true
  git -C "$repo_root" worktree remove --force "$buggy_tree" >/dev/null 2>&1 || true
  rm -rf "$temp_dir"
}
trap cleanup EXIT

git -C "$repo_root" worktree add --detach "$head_tree" "$current_head" >/dev/null
git -C "$repo_root" worktree add --detach "$buggy_tree" "$buggy_commit" >/dev/null

create_temp_test() {
  local worktree="$1"
  cat > "$worktree/$temp_test_rel" <<'EOF'
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlayRegressionEvidenceRejectsIncompleteSemanticAnalysisForm(t *testing.T) {
	home := t.TempDir()
	stdout, stderr, exitCode, err := runInSubprocess(t, home, []string{"play", "tool:success", "Bash"})
	if err == nil {
		t.Fatalf("Run(play incomplete semantic args) error = nil, want exit code 1\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if exitCode != 1 {
		t.Fatalf("Run(play incomplete semantic args) exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	assertContains(t, stderr, "claune: play accepts either <event> or <event> <tool-name> <tool-input>")
	assertContains(t, stderr, "Usage: claune play <event>")
	assertContains(t, stderr, "claune play <event> <tool-name> <tool-input>")
}

func TestPlayRegressionEvidenceBadUsageWinsOverMalformedConfig(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	stdout, stderr, exitCode, err := runInSubprocess(t, home, []string{"play"})
	if err == nil {
		t.Fatalf("Run(play) error = nil, want exit code 1\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if exitCode != 1 {
		t.Fatalf("Run(play) exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	assertContains(t, stderr, "claune: play requires an event")
	assertContains(t, stderr, "Usage: claune play <event>")
	if strings.Contains(stderr, "error loading config") {
		t.Fatalf("stderr = %q, should not contain config load error", stderr)
	}
}
EOF
}

create_temp_test "$head_tree"
create_temp_test "$buggy_tree"

printf '==> Verifying regression test fails on buggy parent %s\n' "$buggy_commit"
set +e
buggy_output="$(cd "$buggy_tree" && PATH="/home/everlier/go/bin:$PATH" go test ./internal/cli -run 'TestPlayRegressionEvidence' 2>&1)"
buggy_status=$?
set -e
printf '%s\n' "$buggy_output"

if [ "$buggy_status" -eq 0 ]; then
  printf 'Expected buggy parent %s to fail the play regression evidence tests, but it passed.\n' "$buggy_commit" >&2
  exit 1
fi

printf '\n==> Verifying the same regression test passes on current HEAD %s\n' "$current_head"
(
	cd "$head_tree"
	PATH="/home/everlier/go/bin:$PATH" go test ./internal/cli -run 'TestPlayRegressionEvidence' 2>&1
)

printf '\nVerified best-possible evidence: the same focused play regression tests fail on %s and pass on %s.\n' "$buggy_commit" "$current_head"
printf 'Limitation: this does not prove the original historical red->green order; it proves the regression/fix relationship is independently reproducible now.\n'
