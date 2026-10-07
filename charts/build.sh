#!/bin/sh
set -eu

# Package the echoheaders chart into this directory and (re)generate the
# Helm repository index served at https://echoheaders.is-a.dev/charts .
#
# On push to main, .github/workflows/static.yaml runs this script and
# publishes index.yaml + *.tgz to GitHub Pages automatically.
#
# Client usage:
#   helm repo add echoheaders https://echoheaders.is-a.dev/charts
#   helm repo update
#   helm upgrade --install echo echoheaders/echoheaders --namespace echoheaders --create-namespace

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
CHART_DIR="$SCRIPT_DIR/../deploy/helm"
REPO_URL="https://echoheaders.is-a.dev/charts"

cd "$SCRIPT_DIR"
rm -f echoheaders-*.tgz

helm package "$CHART_DIR" --destination "$SCRIPT_DIR"
helm repo index "$SCRIPT_DIR" --url "$REPO_URL"

echo "Chart packaged and index.yaml regenerated in $SCRIPT_DIR"
ls -l "$SCRIPT_DIR"
