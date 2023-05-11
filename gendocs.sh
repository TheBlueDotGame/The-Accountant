#!/bin/bash
gomarkdoc ./... > docs.md
cat docs/header.txt docs/git_hub_header.md docs.md docs.md > docs/docs.md
echo "docs generated web page"
cat docs/git_hub_header.md docs.md > README.md
echo "docs generated README.md"
rm docs.md