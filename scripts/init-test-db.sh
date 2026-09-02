#!/bin/bash
# Creates the integration-test database on the container's FIRST boot.
#
# Anything in /docker-entrypoint-initdb.d runs once, when the data directory is
# empty. On later starts the directory has data and this is skipped entirely -
# which is the same rule that makes POSTGRES_DB changes appear to do nothing.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE apimonitor_test;
    GRANT ALL PRIVILEGES ON DATABASE apimonitor_test TO $POSTGRES_USER;
EOSQL

echo "created database apimonitor_test"
