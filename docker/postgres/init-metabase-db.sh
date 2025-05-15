#!/bin/bash
set -e

# Function to check if database exists
database_exists() {
  psql -U "$POSTGRES_USER" -lqt | cut -d \| -f 1 | grep -qw "$1"
}

# Check if the metabase database already exists
if database_exists "metabase"; then
    echo "Database 'metabase' already exists, skipping creation"
else
    echo "Creating 'metabase' database..."
    # Create the metabase database
    psql -U "$POSTGRES_USER" -c "CREATE DATABASE metabase;"
    
    # Grant all privileges to the postgres user
    psql -U "$POSTGRES_USER" -c "GRANT ALL PRIVILEGES ON DATABASE metabase TO $POSTGRES_USER;"
    
    echo "Database 'metabase' created successfully and privileges granted to $POSTGRES_USER"
fi

