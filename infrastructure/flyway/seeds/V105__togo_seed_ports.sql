-- 10 ports for maritime operations (Togo + West African corridor)
\c kinara_port;

INSERT INTO ports (id, port_code, name, country, city, lat, lng, port_type, capacity_teu, is_active, tenant_id, created_at)
VALUES
  (gen_random_uuid(), 'TGLO',  'Port Autonome de Lomé',              'TG', 'Lomé',       6.1375, 1.2647,   'seaport',     1500000, true, 'TG', NOW()),
  (gen_random_uuid(), 'TGANE', 'Port de Pêche d''Aného',             'TG', 'Aného',      6.2300, 1.5900,   'fishing',       15000, true, 'TG', NOW()),
  (gen_random_uuid(), 'GHTE',  'Tema Port',                          'GH', 'Tema',       5.6333, -0.0166,  'seaport',     1200000, true, 'GH', NOW()),
  (gen_random_uuid(), 'BJAN',  'Port Cotonou',                       'BJ', 'Cotonou',    6.3573,  2.4214,  'seaport',      800000, true, 'BJ', NOW()),
  (gen_random_uuid(), 'NGLA',  'Apapa Port Lagos',                   'NG', 'Lagos',      6.4231,  3.3875,  'seaport',     2000000, true, 'NG', NOW()),
  (gen_random_uuid(), 'CIABJ', 'Port d''Abidjan',                    'CI', 'Abidjan',    5.3000, -4.0167,  'seaport',     1100000, true, 'CI', NOW()),
  (gen_random_uuid(), 'SNDK',  'Port de Dakar',                      'SN', 'Dakar',     14.6937, -17.4441, 'seaport',      650000, true, 'SN', NOW()),
  (gen_random_uuid(), 'TGLR',  'Lac de Retenue de Kpimé (Inland)',   'TG', 'Atakpamé',   7.5333,  0.8667,  'inland',         5000, true, 'TG', NOW()),
  (gen_random_uuid(), 'TGMN',  'Terminal Fluvial de Mono',           'TG', 'Agoué',      6.2167,  1.4833,  'river',          3000, true, 'TG', NOW()),
  (gen_random_uuid(), 'GHKD',  'Port Keta (Ghana)',                  'GH', 'Keta',       5.9167,  0.9833,  'fishing',       12000, true, 'GH', NOW())
ON CONFLICT DO NOTHING;

-- Currencies configured across all services
\c kinara_payment;
INSERT INTO currencies (code, name, symbol, exchange_rate_usd, is_active, updated_at)
VALUES
  ('USD', 'US Dollar',           '$',   1.0,    true, NOW()),
  ('XOF', 'West African CFA',    'CFA', 0.00167,true, NOW()),
  ('GHS', 'Ghanaian Cedi',       '₵',   0.0714, true, NOW()),
  ('KES', 'Kenyan Shilling',     'KSh', 0.00769,true, NOW()),
  ('NGN', 'Nigerian Naira',      '₦',   0.000667,true,NOW()),
  ('ETB', 'Ethiopian Birr',      'Br',  0.01786,true, NOW()),
  ('TZS', 'Tanzanian Shilling',  'TSh', 0.000385,true,NOW()),
  ('RWF', 'Rwandan Franc',       'RF',  0.000769,true,NOW()),
  ('EUR', 'Euro',                '€',   1.08,   true, NOW()),
  ('GBP', 'British Pound',       '£',   1.27,   true, NOW())
ON CONFLICT (code) DO UPDATE SET exchange_rate_usd=EXCLUDED.exchange_rate_usd, updated_at=NOW();
