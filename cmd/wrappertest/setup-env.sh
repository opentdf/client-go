#!/usr/bin/env bash
set -euo pipefail

export TDF_USER="user1@nope.com"
export TDF_CLIENTID="tdf-client"
export TDF_KAS_URL="http://localhost:8000"
export TDF_OIDC_URL="http://localhost:51715"
export TDF_ORGNAME="tdf"
export TDF_CLIENTSECRET="123-456"
export TDF_EXTERNALTOKEN=""

./wrappertest
