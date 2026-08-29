-- ── PORTS ──────────────────────────────────────────────────────────────
\c kinara_port;
INSERT INTO ports (id, port_code, name, country, city, port_type, total_berths, annual_capacity_teu, is_operational, created_at, updated_at) VALUES
    (gen_random_uuid(), 'PT-LOME01', 'Port of Lomé',     'TG', 'Lomé',   'seaport', 8, 1800000, true, NOW(), NOW()),
    (gen_random_uuid(), 'PT-ABIDJ1', 'Port of Abidjan',  'CI', 'Abidjan','seaport', 12, 2500000, true, NOW(), NOW()),
    (gen_random_uuid(), 'PT-TEMA01', 'Port of Tema',     'GH', 'Tema',   'seaport', 10, 2000000, true, NOW(), NOW()),
    (gen_random_uuid(), 'PT-LAGOS1', 'Port of Lagos',    'NG', 'Lagos',  'seaport', 20, 5000000, true, NOW(), NOW()),
    (gen_random_uuid(), 'PT-DAKAR1', 'Port of Dakar',    'SN', 'Dakar',  'seaport', 9,  1500000, true, NOW(), NOW());

-- ── COOPERATIVES ────────────────────────────────────────────────────────
\c kinara_cooperative;
INSERT INTO cooperatives (id, name, country, region, primary_commodity, member_count, is_active, created_at, updated_at) VALUES
    (gen_random_uuid(), 'Coopérative Agricole Maritime',  'TG', 'Maritime',  'maize',     45, true, NOW(), NOW()),
    (gen_random_uuid(), 'Union des Producteurs Plateaux', 'TG', 'Plateaux',  'cassava',   62, true, NOW(), NOW()),
    (gen_random_uuid(), 'Coopérative Centrale Vivres',    'TG', 'Centrale',  'yam',       38, true, NOW(), NOW()),
    (gen_random_uuid(), 'Union Agricole du Nord',         'TG', 'Kara',      'sorghum',   51, true, NOW(), NOW()),
    (gen_random_uuid(), 'Coopérative Savanes Céréales',   'TG', 'Savanes',   'millet',    29, true, NOW(), NOW()),
    (gen_random_uuid(), 'Lomé Cotton Producers Union',    'TG', 'Maritime',  'cotton',    83, true, NOW(), NOW()),
    (gen_random_uuid(), 'SOTOCO Soybean Group',           'TG', 'Plateaux',  'soybean',   41, true, NOW(), NOW()),
    (gen_random_uuid(), 'Groundnut Exporters Togo',       'TG', 'Centrale',  'groundnut', 57, true, NOW(), NOW()),
    (gen_random_uuid(), 'Kara Rice Growers Assoc',        'TG', 'Kara',      'rice',      34, true, NOW(), NOW()),
    (gen_random_uuid(), 'Savanes Cowpea Alliance',        'TG', 'Savanes',   'cowpea',    48, true, NOW(), NOW());

-- ── CLINICS ─────────────────────────────────────────────────────────────
\c kinara_clinical;
INSERT INTO facilities (id, name, facility_type, country, region, city, bed_count, is_active, created_at, updated_at) VALUES
    (gen_random_uuid(), 'CHU Sylvanus Olympio',            'hospital',       'TG', 'Maritime', 'Lomé',     350, true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Tsévié',               'hospital',       'TG', 'Maritime', 'Tsévié',   80,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Agoè',            'health_center',  'TG', 'Maritime', 'Lomé',     30,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CHR Atakpamé',                    'hospital',       'TG', 'Plateaux', 'Atakpamé', 120, true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Kpalimé',              'hospital',       'TG', 'Plateaux', 'Kpalimé',  90,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre Médical Notsé',            'health_center',  'TG', 'Plateaux', 'Notsé',    25,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CHR Sokodé',                      'hospital',       'TG', 'Centrale', 'Sokodé',   150, true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Tchamba',              'hospital',       'TG', 'Centrale', 'Tchamba',  60,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Blitta',          'health_center',  'TG', 'Centrale', 'Blitta',   20,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CHR Kara',                        'hospital',       'TG', 'Kara',     'Kara',     180, true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Pagouda',              'hospital',       'TG', 'Kara',     'Pagouda',  50,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre Médical Niamtougou',       'health_center',  'TG', 'Kara',     'Niamtougou',18, true, NOW(), NOW()),
    (gen_random_uuid(), 'CHR Dapaong',                     'hospital',       'TG', 'Savanes',  'Dapaong',  100, true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Mango',                'hospital',       'TG', 'Savanes',  'Mango',    45,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Cinkassé',        'health_center',  'TG', 'Savanes',  'Cinkassé', 15,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital Bon Samaritain Lomé',     'private_clinic', 'TG', 'Maritime', 'Lomé',     40,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Clinique de l''Espoir',           'private_clinic', 'TG', 'Maritime', 'Lomé',     25,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Adidogomé',       'health_center',  'TG', 'Maritime', 'Lomé',     30,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Vogan',                'hospital',       'TG', 'Maritime', 'Vogan',    55,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre Médical Tabligbo',         'health_center',  'TG', 'Maritime', 'Tabligbo', 20,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CHR Atakpamé Sud',                'hospital',       'TG', 'Plateaux', 'Atakpamé', 70,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Badou',                'hospital',       'TG', 'Plateaux', 'Badou',    40,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Amlamé',          'health_center',  'TG', 'Plateaux', 'Amlamé',   18,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Sotouboua',            'hospital',       'TG', 'Centrale', 'Sotouboua',48,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Anié',            'health_center',  'TG', 'Centrale', 'Anié',     22,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Kandé',                'hospital',       'TG', 'Kara',     'Kandé',    38,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre Médical Bassar',           'health_center',  'TG', 'Kara',     'Bassar',   28,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital de Guérin-Kouka',         'hospital',       'TG', 'Kara',     'Guérin-Kouka', 32, true, NOW(), NOW()),
    (gen_random_uuid(), 'Centre de Santé Tone',            'health_center',  'TG', 'Savanes',  'Tone',     15,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Hôpital Préfectoral Oti',         'hospital',       'TG', 'Savanes',  'Sansanné-Mango', 42, true, NOW(), NOW()),
    -- 20 more for full 50
    (gen_random_uuid(), 'CS Lomé Bè',                      'health_center',  'TG', 'Maritime', 'Lomé',     18,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Avedji',                  'health_center',  'TG', 'Maritime', 'Lomé',     22,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Tokoin',                  'health_center',  'TG', 'Maritime', 'Lomé',     20,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Zanguéra',                'health_center',  'TG', 'Maritime', 'Lomé',     16,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Agbalepedogan',           'health_center',  'TG', 'Maritime', 'Lomé',     24,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Kpalimé Nord',                 'health_center',  'TG', 'Plateaux', 'Kpalimé',  18,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Atakpamé Ouest',               'health_center',  'TG', 'Plateaux', 'Atakpamé', 20,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Sokodé Est',                   'health_center',  'TG', 'Centrale', 'Sokodé',   22,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Kara Nord',                    'health_center',  'TG', 'Kara',     'Kara',     15,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Dapaong Est',                  'health_center',  'TG', 'Savanes',  'Dapaong',  18,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Dapaong Ouest',                'health_center',  'TG', 'Savanes',  'Dapaong',  16,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Polyclinique du Golfe',           'private_clinic', 'TG', 'Maritime', 'Lomé',     35,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Clinique Biasa',                  'private_clinic', 'TG', 'Maritime', 'Lomé',     28,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Clinique Fraternité',             'private_clinic', 'TG', 'Maritime', 'Lomé',     22,  true, NOW(), NOW()),
    (gen_random_uuid(), 'Clinique du Lac',                 'private_clinic', 'TG', 'Maritime', 'Lomé',     18,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Tsévié Nord',                  'health_center',  'TG', 'Maritime', 'Tsévié',   20,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Vogan Est',                    'health_center',  'TG', 'Maritime', 'Vogan',    14,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Anécho',                       'health_center',  'TG', 'Maritime', 'Anécho',   19,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Atikoumé',                'health_center',  'TG', 'Maritime', 'Lomé',     21,  true, NOW(), NOW()),
    (gen_random_uuid(), 'CS Lomé Adidogomé Nord',          'health_center',  'TG', 'Maritime', 'Lomé',     17,  true, NOW(), NOW());

-- ── CURRENCIES / WALLET SETUP ───────────────────────────────────────────
\c kinara_payment;
INSERT INTO wallets (id, owner_id, owner_type, currency, balance, status, created_at, updated_at) VALUES
    (gen_random_uuid(), gen_random_uuid(), 'system', 'XOF', 10000000, 'active', NOW(), NOW()),
    (gen_random_uuid(), gen_random_uuid(), 'system', 'USD', 50000,    'active', NOW(), NOW()),
    (gen_random_uuid(), gen_random_uuid(), 'system', 'GHS', 100000,   'active', NOW(), NOW());
