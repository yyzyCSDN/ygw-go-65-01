#!/usr/bin/env sh
set -eu
docker build -f benzhi.Dockerfile -t fnexec-benzhi .
echo "image fnexec-benzhi built"
