#!/usr/bin/env sh
# Builds the API into the project root and runs it. Extra arguments are passed
# to the executable. The .env in the project root is loaded on startup.
set -eu

cd "$(dirname "$0")"

# "email-tracker.exe" on Windows (Git Bash, MSYS), "email-tracker" elsewhere.
bin="email-tracker$(go env GOEXE)"

go build -C ./cmd/api -o "../../$bin"
exec "./$bin" "$@"
