-- Create currencies table with ISO 4217 compliance
-- Spec: docs/specs/003-currency-management.md

BEGIN;

-- Create currencies table
CREATE TABLE IF NOT EXISTS treasury.currencies (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- ISO 4217 fields
    code CHAR(3) NOT NULL,                    -- ISO 4217 alphabetic code (USD, EUR, GBP)
    numeric_code CHAR(3),                      -- ISO 4217 numeric code (840, 978, 826)
    name VARCHAR(100) NOT NULL,                -- Official currency name
    minor_units SMALLINT NOT NULL DEFAULT 2,   -- Decimal places (2 for USD, 0 for JPY)
    
    -- Additional metadata
    symbol VARCHAR(10),                        -- Currency symbol ($, €, £)
    symbol_position VARCHAR(10) DEFAULT 'before', -- before/after amount
    country_codes TEXT[],                      -- Array of ISO 3166 country codes
    is_active BOOLEAN NOT NULL DEFAULT true,   -- Whether currency is currently active
    is_crypto BOOLEAN NOT NULL DEFAULT false,  -- Whether this is a cryptocurrency
    
    -- Status management
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, deprecated, deleted
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Constraints
    CONSTRAINT uk_currencies_code UNIQUE (code),
    CONSTRAINT uk_currencies_numeric_code UNIQUE (numeric_code),
    CONSTRAINT chk_currencies_code_format CHECK (code ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_currencies_numeric_code_format CHECK (numeric_code IS NULL OR numeric_code ~ '^[0-9]{3}$'),
    CONSTRAINT chk_currencies_minor_units CHECK (minor_units >= 0 AND minor_units <= 8),
    CONSTRAINT chk_currencies_status CHECK (status IN ('active', 'inactive', 'deprecated', 'deleted'))
);

-- Indexes for performance
CREATE INDEX idx_currencies_code ON treasury.currencies(code) WHERE status != 'deleted';
CREATE INDEX idx_currencies_numeric_code ON treasury.currencies(numeric_code) 
    WHERE numeric_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_currencies_status ON treasury.currencies(status);
CREATE INDEX idx_currencies_country_codes ON treasury.currencies USING GIN(country_codes);
CREATE INDEX idx_currencies_is_active ON treasury.currencies(is_active) WHERE is_active = true;

-- Trigger for updated_at
CREATE TRIGGER update_currencies_updated_at 
    BEFORE UPDATE ON treasury.currencies
    FOR EACH ROW 
    EXECUTE FUNCTION treasury.update_updated_at_column();

COMMIT;