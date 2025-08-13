-- Migration: 000004_create_financial_institutions_table.up.sql
-- Spec: docs/specs/004-financial-institutions.md

BEGIN;

-- Create financial institutions table
CREATE TABLE IF NOT EXISTS treasury.financial_institutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    swift_code VARCHAR(11),
    iban_prefix VARCHAR(4),
    bank_code VARCHAR(20),
    branch_code VARCHAR(20),
    institution_type VARCHAR(50) NOT NULL,
    country_code CHAR(2) NOT NULL,
    primary_currency CHAR(3),
    street_address_1 VARCHAR(255),
    street_address_2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    phone_number VARCHAR(50),
    fax_number VARCHAR(50),
    email_address VARCHAR(255),
    website_url VARCHAR(255),
    time_zone VARCHAR(50),
    business_hours JSONB,
    holiday_calendar VARCHAR(50),
    regulatory_id VARCHAR(50),
    tax_id VARCHAR(50),
    licenses JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_active BOOLEAN NOT NULL DEFAULT true,
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    suspension_reason TEXT,
    capabilities JSONB,
    notes TEXT,
    external_references JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT uk_institutions_code UNIQUE (code),
    CONSTRAINT uk_institutions_swift UNIQUE (swift_code),
    CONSTRAINT chk_institutions_swift_format CHECK (swift_code IS NULL OR swift_code ~ '^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$'),
    CONSTRAINT chk_institutions_country_format CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_institutions_status CHECK (status IN ('active', 'inactive', 'suspended', 'deleted')),
    CONSTRAINT chk_institutions_type CHECK (institution_type IN ('bank', 'credit_union', 'investment_bank', 'central_bank', 'savings_bank', 'online_bank', 'other'))
);

-- Create indexes
CREATE INDEX idx_institutions_code ON treasury.financial_institutions(code) WHERE status != 'deleted';
CREATE INDEX idx_institutions_swift ON treasury.financial_institutions(swift_code) 
    WHERE swift_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_institutions_country ON treasury.financial_institutions(country_code);
CREATE INDEX idx_institutions_type ON treasury.financial_institutions(institution_type);
CREATE INDEX idx_institutions_status ON treasury.financial_institutions(status);
CREATE INDEX idx_institutions_is_active ON treasury.financial_institutions(is_active) WHERE is_active = true;

-- Add trigger
CREATE TRIGGER update_institutions_updated_at 
    BEFORE UPDATE ON treasury.financial_institutions
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create routing numbers table (supports multiple routing numbers per institution)
CREATE TABLE IF NOT EXISTS treasury.institution_routing_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id) ON DELETE CASCADE,
    routing_number CHAR(9) NOT NULL,
    routing_type VARCHAR(50) NOT NULL DEFAULT 'standard',
    is_primary BOOLEAN NOT NULL DEFAULT false,
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_routing_format CHECK (routing_number ~ '^[0-9]{9}$'),
    CONSTRAINT chk_routing_type CHECK (routing_type IN ('standard', 'wire', 'ach', 'fedwire', 'other')),
    CONSTRAINT uk_routing_number_type UNIQUE (institution_id, routing_number, routing_type)
);

-- Create indexes for routing numbers
CREATE INDEX idx_routing_numbers ON treasury.institution_routing_numbers(routing_number);
CREATE INDEX idx_routing_institution ON treasury.institution_routing_numbers(institution_id);
CREATE INDEX idx_routing_primary ON treasury.institution_routing_numbers(institution_id, is_primary) WHERE is_primary = true;

-- Add trigger for routing numbers updated_at
CREATE TRIGGER update_routing_numbers_updated_at 
    BEFORE UPDATE ON treasury.institution_routing_numbers
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create reference tracking table
CREATE TABLE IF NOT EXISTS treasury.institution_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id),
    table_name VARCHAR(100) NOT NULL,
    column_name VARCHAR(100) NOT NULL,
    reference_count INTEGER NOT NULL DEFAULT 0,
    last_checked TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_institution_references UNIQUE (institution_id, table_name, column_name)
);

-- Insert sample institutions
INSERT INTO treasury.financial_institutions 
    (id, code, name, short_name, swift_code, institution_type, country_code, primary_currency) 
VALUES
    ('a1111111-1111-1111-1111-111111111111', 'JPMORGAN', 'JPMorgan Chase Bank, N.A.', 'Chase', 'CHASUS33', 'bank', 'US', 'USD'),
    ('a2222222-2222-2222-2222-222222222222', 'BOFA', 'Bank of America, N.A.', 'Bank of America', 'BOFAUS3N', 'bank', 'US', 'USD'),
    ('a3333333-3333-3333-3333-333333333333', 'WELLS', 'Wells Fargo Bank, N.A.', 'Wells Fargo', 'WFBIUS6S', 'bank', 'US', 'USD'),
    ('a4444444-4444-4444-4444-444444444444', 'CITI', 'Citibank, N.A.', 'Citibank', 'CITIUS33', 'bank', 'US', 'USD'),
    ('a5555555-5555-5555-5555-555555555555', 'HSBC', 'HSBC Bank USA, N.A.', 'HSBC', 'MRMDUS33', 'bank', 'US', 'USD'),
    ('a6666666-6666-6666-6666-666666666666', 'BARCLAYS', 'Barclays Bank PLC', 'Barclays', 'BARCGB22', 'bank', 'GB', 'GBP'),
    ('a7777777-7777-7777-7777-777777777777', 'DEUTSCHE', 'Deutsche Bank AG', 'Deutsche Bank', 'DEUTDEFF', 'investment_bank', 'DE', 'EUR'),
    ('a8888888-8888-8888-8888-888888888888', 'BNP', 'BNP Paribas SA', 'BNP Paribas', 'BNPAFRPP', 'bank', 'FR', 'EUR');

-- Insert sample routing numbers for US banks
INSERT INTO treasury.institution_routing_numbers 
    (institution_id, routing_number, routing_type, is_primary, description)
VALUES
    -- JPMorgan Chase routing numbers
    ('a1111111-1111-1111-1111-111111111111', '021000021', 'standard', true, 'New York'),
    ('a1111111-1111-1111-1111-111111111111', '322271627', 'standard', false, 'California'),
    ('a1111111-1111-1111-1111-111111111111', '021000021', 'wire', false, 'Wire transfers'),
    
    -- Bank of America routing numbers
    ('a2222222-2222-2222-2222-222222222222', '026009593', 'wire', true, 'Wire transfers'),
    ('a2222222-2222-2222-2222-222222222222', '121000358', 'standard', false, 'California'),
    ('a2222222-2222-2222-2222-222222222222', '051000017', 'standard', false, 'Virginia'),
    
    -- Wells Fargo routing numbers
    ('a3333333-3333-3333-3333-333333333333', '121000248', 'standard', true, 'California'),
    ('a3333333-3333-3333-3333-333333333333', '121042882', 'wire', false, 'Wire transfers'),
    ('a3333333-3333-3333-3333-333333333333', '102000076', 'standard', false, 'Colorado'),
    
    -- Citibank routing numbers
    ('a4444444-4444-4444-4444-444444444444', '021000089', 'standard', true, 'New York'),
    ('a4444444-4444-4444-4444-444444444444', '321171184', 'standard', false, 'California'),
    ('a4444444-4444-4444-4444-444444444444', '021000089', 'wire', false, 'Wire transfers'),
    
    -- HSBC routing numbers
    ('a5555555-5555-5555-5555-555555555555', '021001088', 'standard', true, 'New York'),
    ('a5555555-5555-5555-5555-555555555555', '021001088', 'wire', false, 'Wire transfers');

COMMIT;