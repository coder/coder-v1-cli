#!/bin/bash
echo "Formatting..."

go mod tidy
gofmt -w -s .
goimports -w "-local=$$(go list -m)" .

if [ "$CI" != "" ]; then
  if [[ $(git ls-files --other --modified --exclude-standard) != "" ]]; then
    echo "Files need generation or are formatted incorrectly:"
    git -c color.ui=always status | grep --color=no '\e\[31m'
    echo "Please run the following locally:"
    echo "  ./ci/steps/fmt.sh"
    exit 1
  fi
fi