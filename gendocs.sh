#!/bin/bash
gomarkdoc ./... > docs.md
cat header.md docs.md > README.md
echo "docs generated README.md"
rm docs.md
