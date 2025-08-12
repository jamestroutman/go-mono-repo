-- Create treasury database and user
CREATE DATABASE treasury_db;
CREATE USER treasury_user WITH PASSWORD 'treasury_pass';
GRANT ALL PRIVILEGES ON DATABASE treasury_db TO treasury_user;

-- Grant schema permissions
\c treasury_db
GRANT ALL ON SCHEMA public TO treasury_user;
