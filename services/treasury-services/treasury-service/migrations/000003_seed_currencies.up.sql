-- Seed initial ISO 4217 currencies
-- Spec: docs/specs/003-currency-management.md

BEGIN;

-- Insert common world currencies
INSERT INTO treasury.currencies (code, numeric_code, name, minor_units, symbol, country_codes, is_active, status, activated_at, created_by) VALUES
    ('USD', '840', 'United States Dollar', 2, '$', ARRAY['US', 'AS', 'EC', 'GU', 'MH', 'FM', 'MP', 'PW', 'PR', 'TC', 'VI', 'UM'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('EUR', '978', 'Euro', 2, '€', ARRAY['AD', 'AT', 'BE', 'CY', 'EE', 'FI', 'FR', 'DE', 'GR', 'IE', 'IT', 'XK', 'LV', 'LT', 'LU', 'MT', 'MC', 'ME', 'NL', 'PT', 'SM', 'SK', 'SI', 'ES', 'VA'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('GBP', '826', 'Pound Sterling', 2, '£', ARRAY['GB', 'IM', 'JE', 'GG'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('JPY', '392', 'Japanese Yen', 0, '¥', ARRAY['JP'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('CHF', '756', 'Swiss Franc', 2, 'CHF', ARRAY['CH', 'LI'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('CAD', '124', 'Canadian Dollar', 2, '$', ARRAY['CA'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('AUD', '036', 'Australian Dollar', 2, '$', ARRAY['AU', 'CX', 'CC', 'HM', 'KI', 'NR', 'NF', 'TV'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('CNY', '156', 'Chinese Yuan', 2, '¥', ARRAY['CN'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('SEK', '752', 'Swedish Krona', 2, 'kr', ARRAY['SE'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('NZD', '554', 'New Zealand Dollar', 2, '$', ARRAY['NZ', 'CK', 'NU', 'PN', 'TK'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('MXN', '484', 'Mexican Peso', 2, '$', ARRAY['MX'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('SGD', '702', 'Singapore Dollar', 2, '$', ARRAY['SG'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('HKD', '344', 'Hong Kong Dollar', 2, '$', ARRAY['HK'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('NOK', '578', 'Norwegian Krone', 2, 'kr', ARRAY['NO', 'SJ', 'BV'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('KRW', '410', 'South Korean Won', 0, '₩', ARRAY['KR'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('TRY', '949', 'Turkish Lira', 2, '₺', ARRAY['TR'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('RUB', '643', 'Russian Ruble', 2, '₽', ARRAY['RU'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('INR', '356', 'Indian Rupee', 2, '₹', ARRAY['IN', 'BT'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('BRL', '986', 'Brazilian Real', 2, 'R$', ARRAY['BR'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('ZAR', '710', 'South African Rand', 2, 'R', ARRAY['ZA', 'LS', 'NA', 'SZ'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('DKK', '208', 'Danish Krone', 2, 'kr', ARRAY['DK', 'FO', 'GL'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('PLN', '985', 'Polish Zloty', 2, 'zł', ARRAY['PL'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('THB', '764', 'Thai Baht', 2, '฿', ARRAY['TH'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('IDR', '360', 'Indonesian Rupiah', 2, 'Rp', ARRAY['ID'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('CZK', '203', 'Czech Koruna', 2, 'Kč', ARRAY['CZ'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('ILS', '376', 'Israeli New Shekel', 2, '₪', ARRAY['IL', 'PS'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('AED', '784', 'UAE Dirham', 2, 'AED', ARRAY['AE'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('PHP', '608', 'Philippine Peso', 2, '₱', ARRAY['PH'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('SAR', '682', 'Saudi Riyal', 2, 'SAR', ARRAY['SA'], true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('MYR', '458', 'Malaysian Ringgit', 2, 'RM', ARRAY['MY'], true, 'active', CURRENT_TIMESTAMP, 'system');

-- Insert major cryptocurrencies (no ISO numeric codes)
INSERT INTO treasury.currencies (code, name, minor_units, symbol, is_crypto, is_active, status, activated_at, created_by) VALUES
    ('BTC', 'Bitcoin', 8, '₿', true, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('ETH', 'Ethereum', 8, 'Ξ', true, true, 'active', CURRENT_TIMESTAMP, 'system');  -- Note: ETH actually has 18 decimals but limited to 8 by constraint

-- Insert special ISO codes
INSERT INTO treasury.currencies (code, numeric_code, name, minor_units, is_active, status, activated_at, created_by) VALUES
    ('XAU', '959', 'Gold (Troy Ounce)', 0, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('XAG', '961', 'Silver (Troy Ounce)', 0, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('XPT', '962', 'Platinum (Troy Ounce)', 0, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('XPD', '964', 'Palladium (Troy Ounce)', 0, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('XTS', '963', 'Code for Testing', 2, true, 'active', CURRENT_TIMESTAMP, 'system'),
    ('XXX', '999', 'No Currency', 0, true, 'active', CURRENT_TIMESTAMP, 'system');

COMMIT;