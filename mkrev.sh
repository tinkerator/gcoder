#!/bin/bash
# usage: ./version.sh [major-version]

MAJOR="${1}"

# Only works when tree is fully committed.
test -z "$(git status -s)"
if [[ ${?} -ne 0 ]]; then
    echo "commit work first"
    exit 1
fi

LAST_TAG="$(git tag -l --sort=committerdate 'v*'|tail -1)"
if [[ -z "${LAST_TAG}" ]]; then
    echo "no base tag to work with, using v0.0.0"
    LAST_TAG="v0.0.0"
fi

BITS=(${LAST_TAG//./ })
if [[ -n "${MAJOR}" ]]; then
    MINOR="0"
else
    MAJOR="${BITS[1]}"
    MINOR=$((${BITS[2]}+1))
fi
NEW_TAG="${BITS[0]}.${MAJOR}.${MINOR}"
cat > revision.go <<EOF
package gcoder

var gCoderRevision = "zappem.net/pub/io/gcoder ${NEW_TAG}"
EOF
git add revision.go
git commit -s -m "Set revision to ${NEW_TAG}"
git tag "${NEW_TAG}"
echo "created ${NEW_TAG}"
