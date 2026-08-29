-- 5 agricultural cooperatives in Togo
\c kinara_cooperative;

INSERT INTO cooperatives (id, coop_ref, name, phone, region, country, primary_crop, member_count, is_active, tenant_id, created_at, updated_at)
VALUES
  (gen_random_uuid(), 'COOP-MARITIME', 'Coopérative Agricole du Maritime',   '+22891001001', 'Maritime', 'TG', 'maize',   85,  true, 'TG', NOW(), NOW()),
  (gen_random_uuid(), 'COOP-PLATEAUX', 'Coopérative des Plateaux Verts',     '+22891001002', 'Plateaux', 'TG', 'cocoa',   120, true, 'TG', NOW(), NOW()),
  (gen_random_uuid(), 'COOP-CENTRALE', 'Union des Agriculteurs de Centrale', '+22891001003', 'Centrale', 'TG', 'yam',     65,  true, 'TG', NOW(), NOW()),
  (gen_random_uuid(), 'COOP-KARA',     'Coopérative Kara-Nord',              '+22891001004', 'Kara',     'TG', 'sorghum', 95,  true, 'TG', NOW(), NOW()),
  (gen_random_uuid(), 'COOP-SAVANES',  'Union Paysanne des Savanes',         '+22891001005', 'Savanes',  'TG', 'millet',  110, true, 'TG', NOW(), NOW())
ON CONFLICT DO NOTHING;
