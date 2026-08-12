#!/bin/sh
set -e

# Start the official PostgreSQL entrypoint in the background so initdb and any
# init scripts run as usual, then keep us in charge of this process.
/usr/local/bin/docker-entrypoint.sh postgres &
pg_pid=$!

# Forward stop signals so `docker stop` shuts postgres down gracefully.
trap 'kill -TERM $pg_pid 2>/dev/null; wait $pg_pid 2>/dev/null; exit 0' TERM INT

# The superuser role is named after POSTGRES_USER (default "postgres"); this
# deployment sets POSTGRES_USER=messenger, so there is no "postgres" role.
SUPERUSER="${POSTGRES_USER:-postgres}"

# Wait for the real server (not the temp one used during first-boot init) to
# accept local connections on the default socket directory.
until pg_isready -h /var/run/postgresql -U "${SUPERUSER}" -q 2>/dev/null; do
  sleep 1
done

# POSTGRES_PASSWORD is only applied by the official image when the data volume
# is empty. On an already-initialized volume the password stays whatever it was
# created with, so a changed DB_PASSWORD breaks backend auth. Fix it on every
# start by re-syncing the app user's password to POSTGRES_PASSWORD.
# Local socket connections use trust auth, so SUPERUSER needs no password here.
# psql variables :"name" / :'pw' are escaped by psql, so special characters in
# the password (or user) are safe.
if [ -n "${POSTGRES_PASSWORD}" ]; then
  psql -h /var/run/postgresql -U "${SUPERUSER}" -d postgres -v ON_ERROR_STOP=1 \
    -v uname="${POSTGRES_USER:-messenger}" -v pw="${POSTGRES_PASSWORD}" \
    -c "ALTER USER :\"uname\" WITH PASSWORD :'pw';" >/dev/null 2>&1 || \
    echo "[entrypoint] WARNING: could not sync password for user '${POSTGRES_USER:-messenger}'" >&2
fi

wait $pg_pid