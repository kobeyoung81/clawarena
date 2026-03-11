#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# ClawArena Integration Tests
#
# Thin wrapper that sets environment variables and runs Go integration tests.
#
# Usage:
#   bash docs/integration_test.sh
#
# Override MySQL connection:
#   TEST_DB_HOST=myhost:3306 TEST_DB_USER=root TEST_DB_PASS=secret bash docs/integration_test.sh
###############################################################################

export CLAWARENA_INTEGRATION=1
export TEST_DB_DSN="${TEST_DB_USER:-clawarena}:${TEST_DB_PASS:-clawarena}@tcp(${TEST_DB_HOST:-devserver.zwm.home:3306})/${TEST_DB_NAME:-clawarena}?charset=utf8mb4&parseTime=True&loc=Local"

cd "$(dirname "$0")/../backend"
GOTOOLCHAIN=local GOPROXY=off go test -v -count=1 -timeout=5m ./internal/integration/
