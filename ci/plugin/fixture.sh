#!/bin/sh
# Local-directory marketplaces are live development folders in Claude. Exercise
# the Git install path using a process-local URL rewrite to a disposable repository.
# No network or user Git configuration is changed.
install_plugin_fixture() {
  fixture_root="$1"
  fixture_work="$2"
  export CLAUDE_CONFIG_DIR="$fixture_work/config"
  mkdir -p "$CLAUDE_CONFIG_DIR" "$fixture_work/workspace"
  fixture_mode="${3:-stage}"
  go run "$fixture_root/ci/plugin/main.go" "$fixture_mode" "$fixture_root" "$fixture_work/marketplace"
  claude plugin validate "$fixture_work/marketplace/.claude-plugin/plugin.json" --strict --json
  git -C "$fixture_work/marketplace" init -q --template=
  git -C "$fixture_work/marketplace" -c core.excludesFile=/dev/null -c core.autocrlf=false add --all
  git -C "$fixture_work/marketplace" -c user.name='Plugin acceptance test' \
    -c user.email='plugin-test@example.invalid' -c commit.gpgsign=false \
    -c core.hooksPath=/dev/null commit -qm 'Plugin fixture'
  (
    cd "$fixture_work/workspace"
    export GIT_CONFIG_COUNT=1
    export GIT_CONFIG_KEY_0="url.file://$fixture_work/marketplace.insteadOf"
    export GIT_CONFIG_VALUE_0=https://autopi-plugin-test.invalid/marketplace.git
    claude plugin install automation-pilot-skills --marketplace https://autopi-plugin-test.invalid/marketplace.git --scope user --json
  )
  rm -rf "$fixture_work/marketplace"
  (
    cd "$fixture_work/workspace"
    claude plugin details automation-pilot-skills > "$fixture_work/inventory.txt"
    grep -Fq 'Skills (12)' "$fixture_work/inventory.txt"
  )
}
