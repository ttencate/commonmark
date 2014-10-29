#!/bin/bash

# Go to the project root directory (one level up from the location of this script)
cd $(dirname "$0")/..

# Fetch the latest spec
curl -o spec.txt 'https://raw.githubusercontent.com/jgm/CommonMark/master/spec.txt'
