# MooveIt Separated VM Setup

This guide explains how to set up MooveIt backend services across two VMs to resolve resource sharing issues.

## Architecture Overview

The system is split across two VMs:

1. **VM1**: Database (PostgreSQL) + Metabase
   - PostgreSQL server with exposed port for external access
   - Metabase analytics platform
   
2. **VM2**: Go Application
   - Your main MooveIt Go web server
   - Connects to the database on VM1

## Prerequisites

- Two VM instances with Docker and Docker Compose installed
- Network connectivity between the two VMs (same VPC or proper networking rules)
- Git installed on VM2 to clone the repository

## VM1 Setup (Database + Metabase)

1. **Create the VM instance**:
   - Recommended: 2+ CPU cores, 4+ GB RAM
   - Ubuntu 20.04 LTS or similar

2. **Configure firewall rules**:
   - Allow inbound traffic on port 5432 (PostgreSQL) from VM2's IP
   - Allow inbound traffic on port 3000 (Metabase) if you need to access Metabase directly

3. **Prepare the environment**:
   ```bash
   # Create a directory for the project
   mkdir -p mooveit-db
   cd mooveit-db
   
   # Copy docker-compose file and environment variables
   # (Upload the docker-compose.db-metabase.yml and .env.vm1 files)
   
   # Rename files
   mv docker-compose.db-metabase.yml docker-compose.yml
   mv .env.vm1 .env
   
   # Create the required directory structure
   mkdir -p docker/postgres
   ```

4. **Create the init script**:
   ```bash
   # Create the initialization script for the Metabase database
   cat > docker/postgres/init-metabase-db.sh << 'EOF'
   #!/bin/bash
   set -e
   
   psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
     CREATE DATABASE metabase;
     GRANT ALL PRIVILEGES ON DATABASE metabase TO $POSTGRES_USER;
   EOSQL
   EOF
   
   # Make the script executable
   chmod +x docker/postgres/init-metabase-db.sh
   ```

5. **Start the services**:
   ```bash
   docker-compose up -d
   ```

6. **Note the VM1's IP address**:
   ```bash
   # Get the VM's IP address
   hostname -I | awk '{print $1}'
   ```
   Save this IP address as you'll need it for VM2 configuration.

## VM2 Setup (Go Application)

1. **Create the VM instance**:
   - Recommended: 2+ CPU cores, 4+ GB RAM
   - Ubuntu 20.04 LTS or similar

2. **Configure firewall rules**:
   - Allow inbound traffic on port 8080 (Go application)

3. **Clone the repository**:
   ```bash
   git clone https://github.com/your-repo/mooveit-backend.git
   cd mooveit-backend
   
   # Copy the Docker Compose file and environment variables
   # (Upload the docker-compose.go-app.yml and .env.vm2 files)
   
   # Rename files
   mv docker-compose.go-app.yml docker-compose.yml
   mv .env.vm2 .env
   ```

4. **Update the environment file**:
   Edit the `.env` file and replace placeholders:
   - `VM1_IP_ADDRESS`: The IP address of VM1 noted earlier
   - `VM2_IP_ADDRESS`: The IP address of the current VM (VM2)

   ```bash
   # Get VM2's IP address
   hostname -I | awk '{print $1}'
   
   # Edit the .env file
   nano .env
   ```

5. **Start the Go application**:
   ```bash
   docker-compose up -d
   ```

## Testing the Setup

1. **Test database connectivity**:
   On VM2, try connecting to the database:
   ```bash
   docker exec -it go-app ping -c 3 VM1_IP_ADDRESS
   ```

2. **Test the Go application**:
   Access the API at `http://VM2_IP_ADDRESS:8080`

3. **Test Metabase**:
   Access Metabase at `http://VM1_IP_ADDRESS:3000`

## Troubleshooting

- **Database connection issues**:
  - Verify PostgreSQL is running on VM1: `docker-compose ps`
  - Check that port 5432 is open between VMs: `telnet VM1_IP_ADDRESS 5432`
  - Verify credentials in the .env file on VM2

- **Application startup issues**:
  - Check application logs: `docker-compose logs -f go-app`
  - Verify all environment variables are properly set

## Backup and Maintenance

1. **Database Backups**:
   ```bash
   # On VM1
   docker exec postgres-db pg_dump -U postgres mooveit > backup_$(date +%Y%m%d).sql
   ```

2. **Updating the Go Application**:
   ```bash
   # On VM2
   git pull
   docker-compose down
   docker-compose up -d --build
   ```

## Security Considerations

- Use private networking between VMs when possible
- Consider using SSH tunneling instead of exposing PostgreSQL directly
- Set up proper firewall rules to allow only necessary traffic
- Use strong passwords for database access
- Consider using environment-specific secrets management

