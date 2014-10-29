#!/bin/bash

# Go to the project root directory (one level up from the location of this script)
cd $(dirname "$0")/..

# Install Git hooks
ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
