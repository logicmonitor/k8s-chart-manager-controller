#!/bin/bash

set -e

GOPACKAGES=$(go list ./... | grep -v /vendor/)
GOFILES=$(find . -type f -name '*.go' -not -path "./vendor/*")

test_packages() {
  echo "Testing packages"
  local coverage_report="coverage.txt"
  local profile="profile.out"
  if [[ -f ${coverage_report} ]]; then
    rm ${coverage_report}
  fi
  touch ${coverage_report}
  for package in ${GOPACKAGES[@]}; do
    go test -v -race -coverprofile=${profile} -covermode=atomic $package
    if [ -f ${profile} ]; then
      cat ${profile} >> ${coverage_report}
      rm ${profile}
    fi
  done
}

lint_packages() {
  echo "Linting packages"
  gometalinter --aggregate --vendor --vendored-linters --enable-gc --exclude="zz_generated" --enable-all --disable=gas --disable=gotype --disable=lll --disable=safesql --deadline=1200s --disable=unparam ./...
}

format_files() {
  echo "Formatting files"
  local gofmtfiles="$(gofmt -l -d -s ${GOFILES})"
  if [ ! -z "${gofmtfiles}" ]; then
    echo -e "Failed gofmt files:\n${gofmtfiles}"
    exit 1
  fi
}

if [ "$CI" == "true" ]; then
  lint_packages
  format_files
  test_packages
else
  echo "Running in non-CI environment. Skipping tests."
  touch coverage.txt
fi

exit 0
