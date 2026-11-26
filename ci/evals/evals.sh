#!/bin/sh
set -eu
eval_repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$eval_repository/ci/evals"
exec go run ./runner "$@"
