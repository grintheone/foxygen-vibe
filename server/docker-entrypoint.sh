#!/bin/sh
set -eu

/app/bootstrap
exec /app/server
