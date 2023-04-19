#!/bin/bash

gomarkdoc ./... > docs.md
cat docs/header.txt docs.md > docs/docs.md
rm docs.md
echo "docs generated"
