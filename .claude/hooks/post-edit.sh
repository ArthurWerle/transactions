#!/usr/bin/env bash
# PostToolUse: gofmt the edited Go file. Scoped to that one file; silent when gofmt is missing.
set -u
command -v jq >/dev/null 2>&1 || exit 0
f=$(jq -r '.tool_input.file_path // empty')
case "$f" in *.go) ;; *) exit 0 ;; esac
[ -f "$f" ] || exit 0
command -v gofmt >/dev/null 2>&1 || exit 0
if ! out=$(gofmt -s -w "$f" 2>&1); then
  echo "gofmt failed on $f (syntax error?):" >&2
  echo "$out" >&2
  exit 2
fi
exit 0
