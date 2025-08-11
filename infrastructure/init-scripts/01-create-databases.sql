-- Initialize databases for the monorepo services
-- This script runs automatically when PostgreSQL container starts for the first time

-- Create treasury database and user
-- Spec: services/treasury-services/treasury-service/docs/specs/001-database-connection.md
CREATE DATABASE treasury_db;
CREATE USER treasury_user WITH PASSWORD 'treasury_pass';
GRANT ALL PRIVILEGES ON DATABASE treasury_db TO treasury_user;

-- Grant schema permissions
\c treasury_db
GRANT ALL ON SCHEMA public TO treasury_user;

-- Return to default database
\c postgres

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully';
    RAISE NOTICE 'Created database: treasury_db';
    RAISE NOTICE 'Created user: treasury_user';
END $$;