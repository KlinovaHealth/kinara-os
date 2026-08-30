-- V049: Togo Pilot Seed Data (schema-aligned)
-- Seeds 100 clinics, 1000 patients, 500 farmers, 10 ports, FX rates,
-- 10 cooperatives, and initial market prices for the Oct 2026 Togo pilot.
-- All inserts are idempotent (ON CONFLICT DO NOTHING).
--
-- PHI columns (full_name_enc, phone_enc, national_id_enc, etc.) use
-- SHA-256 placeholder ciphertext. Real AES-256-GCM ciphertext must be
-- written by the application layer — these values satisfy NOT NULL
-- constraints and make the schema testable without real PHI.

-- ═══════════════════════════════════════════════════════
-- SECTION 0: kinara_patient — create clinics registry table
-- clinics do not have their own service migration; they are anchored
-- in kinara_patient as the primary health-facility reference entity.
-- ═══════════════════════════════════════════════════════
\c kinara_patient;

CREATE TABLE IF NOT EXISTS clinics (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    phone          TEXT,
    address        TEXT,
    region         TEXT,
    country        TEXT        NOT NULL,
    clinic_type    TEXT        NOT NULL DEFAULT 'health_center'
                   CHECK (clinic_type IN (
                       'health_center','district','prefectoral','regional',
                       'university','dispensary','health_post','private',
                       'community','maternity'
                   )),
    capacity_beds  INT         NOT NULL DEFAULT 0,
    is_active      BOOLEAN     NOT NULL DEFAULT true,
    tenant_id      TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clinics_country_region ON clinics(country, region);
CREATE INDEX IF NOT EXISTS idx_clinics_tenant         ON clinics(tenant_id);

-- ═══════════════════════════════════════════════════════
-- SECTION 1: kinara_patient — 100 Togolese clinic sites
-- ═══════════════════════════════════════════════════════

INSERT INTO clinics (id, name, phone, address, region, country, clinic_type, capacity_beds, is_active, tenant_id, created_at) VALUES
  (gen_random_uuid(), 'Centre de Santé de Lomé-Nord',         '+22890001001', '12 Rue des Cliniques, Lomé',              'Maritime',  'TG', 'health_center',  30, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District de Tsévié',        '+22890001002', '5 Avenue de la Santé, Tsévié',            'Maritime',  'TG', 'district',       80, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kpalimé',           '+22890001003', '23 Rue du Commerce, Kpalimé',             'Plateaux',  'TG', 'health_center',  25, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital Préfectoral de Sokodé',        '+22890001004', '8 Boulevard Central, Sokodé',             'Centrale',  'TG', 'prefectoral',   120, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kara',              '+22890001005', '45 Rue de l''Hôpital, Kara',              'Kara',      'TG', 'regional',      200, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Vogan',                 '+22890001006', '3 Place du Marché, Vogan',                'Maritime',  'TG', 'dispensary',     10, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé d''Aného',             '+22890001007', '7 Rue de la Mer, Aného',                  'Maritime',  'TG', 'health_center',  20, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District d''Atakpamé',      '+22890001008', '15 Avenue de la République, Atakpamé',    'Plateaux',  'TG', 'district',       60, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre Médical de Badou',              '+22890001009', '2 Rue Principale, Badou',                 'Plateaux',  'TG', 'health_center',  18, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Notsé',                 '+22890001010', '9 Rue du Dispensaire, Notsé',             'Plateaux',  'TG', 'dispensary',     12, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Tchamba',           '+22890001011', '6 Rue de la Santé, Tchamba',              'Centrale',  'TG', 'health_center',  22, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital Régional de Sokodé',           '+22890001012', '1 Boulevard du Nord, Sokodé',             'Centrale',  'TG', 'regional',      150, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Bassar',            '+22890001013', '14 Rue Bassar, Bassar',                   'Centrale',  'TG', 'health_center',  16, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Bafilo',            '+22890001014', '11 Avenue Centrale, Bafilo',              'Kara',      'TG', 'health_center',  20, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Niamtougou',            '+22890001015', '4 Rue du Village, Niamtougou',            'Kara',      'TG', 'dispensary',      8, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District de Kara',          '+22890001016', '22 Avenue de l''Hôpital, Kara',           'Kara',      'TG', 'district',       90, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Pagouda',           '+22890001017', '5 Rue Principale, Pagouda',               'Kara',      'TG', 'health_center',  14, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kandé',             '+22890001018', '8 Rue du Marché, Kandé',                  'Kara',      'TG', 'health_center',  12, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital Régional de Dapaong',          '+22890001019', '3 Boulevard Principal, Dapaong',           'Savanes',   'TG', 'regional',      180, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Tone',              '+22890001020', '17 Rue de Tone, Dapaong',                 'Savanes',   'TG', 'health_center',  20, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Mandouri',              '+22890001021', '1 Rue du Dispensaire, Mandouri',           'Savanes',   'TG', 'dispensary',      6, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Cinkassé',          '+22890001022', '9 Avenue Principale, Cinkassé',            'Savanes',   'TG', 'health_center',  15, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Tandjouaré',        '+22890001023', '6 Rue de la Santé, Tandjouaré',           'Savanes',   'TG', 'health_center',  14, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District de Lomé-Sud',      '+22890001024', '30 Boulevard du Littoral, Lomé',           'Maritime',  'TG', 'district',       70, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Zio',               '+22890001025', '12 Rue Zio, Tsévié',                      'Maritime',  'TG', 'health_center',  18, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Kouvé',                 '+22890001026', '4 Rue Principale, Kouvé',                 'Maritime',  'TG', 'dispensary',      8, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Tabligbo',          '+22890001027', '7 Avenue de la Santé, Tabligbo',           'Maritime',  'TG', 'health_center',  16, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital Universitaire de Lomé',        '+22890001028', '1 Rue de l''Université, Lomé',             'Maritime',  'TG', 'university',    350, true, 'TG', NOW()),
  (gen_random_uuid(), 'CHU Campus de Lomé',                   '+22890001029', '44 Avenue du CHU, Lomé',                  'Maritime',  'TG', 'university',    400, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kévé',              '+22890001030', '5 Rue de Kévé, Kévé',                     'Maritime',  'TG', 'health_center',  15, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Amou-Oblo',             '+22890001031', '2 Rue Principale, Amou-Oblo',             'Plateaux',  'TG', 'dispensary',      7, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé d''Elavagnon',         '+22890001032', '8 Avenue d''Elavagnon, Elavagnon',         'Plateaux',  'TG', 'health_center',  13, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital Préfectoral d''Amou',          '+22890001033', '3 Rue de l''Hôpital, Amou-Oblo',          'Plateaux',  'TG', 'prefectoral',    55, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Glei',              '+22890001034', '6 Rue Principale, Glei',                  'Plateaux',  'TG', 'health_center',  11, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Yikpa',                 '+22890001035', '1 Rue du Village, Yikpa',                 'Plateaux',  'TG', 'dispensary',      5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Sotouboua',         '+22890001036', '9 Boulevard de Sotouboua, Sotouboua',      'Centrale',  'TG', 'health_center',  19, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Blitta',                '+22890001037', '3 Rue Principale, Blitta',                'Centrale',  'TG', 'dispensary',      9, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kabou',             '+22890001038', '7 Rue de Kabou, Kabou',                   'Centrale',  'TG', 'health_center',  12, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District de Tchamba',       '+22890001039', '2 Avenue Principale, Tchamba',             'Centrale',  'TG', 'district',       65, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Dankpen',           '+22890001040', '5 Rue de Dankpen, Kara',                  'Kara',      'TG', 'health_center',  16, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Défalé',                '+22890001041', '4 Rue du Dispensaire, Défalé',            'Kara',      'TG', 'dispensary',      6, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Guérin-Kouka',      '+22890001042', '1 Rue Principale, Guérin-Kouka',           'Kara',      'TG', 'health_center',  14, true, 'TG', NOW()),
  (gen_random_uuid(), 'Hôpital de District de Bassar',        '+22890001043', '8 Boulevard de Bassar, Bassar',            'Kara',      'TG', 'district',       70, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Péssidé',           '+22890001044', '3 Rue du Centre, Péssidé',                'Kara',      'TG', 'health_center',  10, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Naki-Ouest',        '+22890001045', '6 Rue Principale, Naki-Ouest',            'Savanes',   'TG', 'health_center',  12, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Korbongou',             '+22890001046', '2 Rue du Village, Korbongou',             'Savanes',   'TG', 'dispensary',      5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Tambimong',         '+22890001047', '9 Rue Principale, Tambimong',             'Savanes',   'TG', 'health_center',  11, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Bombouaka',         '+22890001048', '4 Rue de Bombouaka, Bombouaka',           'Savanes',   'TG', 'health_center',  13, true, 'TG', NOW()),
  (gen_random_uuid(), 'Dispensaire de Oti',                   '+22890001049', '1 Rue du Dispensaire, Oti',               'Savanes',   'TG', 'dispensary',      6, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Bogou',             '+22890001050', '7 Rue Principale, Bogou',                 'Savanes',   'TG', 'health_center',  10, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Dévégo',             '+22890001051', 'Dévégo, Maritime',                        'Maritime',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Hahotoé',            '+22890001052', 'Hahotoé, Maritime',                       'Maritime',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Baguida',            '+22890001053', 'Baguida, Maritime',                       'Maritime',  'TG', 'health_post',     5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Togblékopé',         '+22890001054', 'Togblékopé, Maritime',                    'Maritime',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Gboto',              '+22890001055', 'Gboto, Maritime',                         'Maritime',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Dalave',             '+22890001056', 'Dalave, Maritime',                        'Maritime',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Dzolo',              '+22890001057', 'Dzolo, Maritime',                         'Maritime',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kpémé',              '+22890001058', 'Kpémé, Maritime',                         'Maritime',  'TG', 'health_post',     5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Anfoin',             '+22890001059', 'Anfoin, Maritime',                        'Maritime',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Sévagan',            '+22890001060', 'Sévagan, Maritime',                       'Maritime',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Gati',               '+22890001061', 'Gati, Plateaux',                          'Plateaux',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Wahala',             '+22890001062', 'Wahala, Plateaux',                        'Plateaux',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Tomégbé',            '+22890001063', 'Tomégbé, Plateaux',                       'Plateaux',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Tsévié-Nord',        '+22890001064', 'Tsévié-Nord, Maritime',                   'Maritime',  'TG', 'health_post',     5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Lomé-Est',           '+22890001065', 'Quartier Bè, Lomé',                       'Maritime',  'TG', 'health_post',     6, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Lomé-Ouest',         '+22890001066', 'Quartier Aflao, Lomé',                    'Maritime',  'TG', 'health_post',     5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kpondavé',           '+22890001067', 'Kpondavé, Centrale',                      'Centrale',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Alibi',              '+22890001068', 'Alibi, Centrale',                         'Centrale',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Tchébébé',           '+22890001069', 'Tchébébé, Centrale',                      'Centrale',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Yéloum',             '+22890001070', 'Yéloum, Centrale',                        'Centrale',  'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kpodjin',            '+22890001071', 'Kpodjin, Centrale',                       'Centrale',  'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Galangashi',         '+22890001072', 'Galangashi, Kara',                        'Kara',      'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Pagala',             '+22890001073', 'Pagala, Kara',                            'Kara',      'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kamboli',            '+22890001074', 'Kamboli, Kara',                           'Kara',      'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Djamde',             '+22890001075', 'Djamde, Kara',                            'Kara',      'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Assoukoko',          '+22890001076', 'Assoukoko, Kara',                         'Kara',      'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Barkoissi',          '+22890001077', 'Barkoissi, Savanes',                      'Savanes',   'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Gando',              '+22890001078', 'Gando, Savanes',                          'Savanes',   'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Koundjoaré',         '+22890001079', 'Koundjoaré, Savanes',                     'Savanes',   'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Margba',             '+22890001080', 'Margba, Savanes',                         'Savanes',   'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Nagbeni',            '+22890001081', 'Nagbeni, Savanes',                        'Savanes',   'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Sadori',             '+22890001082', 'Sadori, Savanes',                         'Savanes',   'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Nioukpourma',        '+22890001083', 'Nioukpourma, Savanes',                    'Savanes',   'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Dapango-Nord',       '+22890001084', 'Dapango-Nord, Savanes',                   'Savanes',   'TG', 'health_post',     5, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Namoundjoga',        '+22890001085', 'Namoundjoga, Savanes',                    'Savanes',   'TG', 'health_post',     4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Bombou',             '+22890001086', 'Bombou, Savanes',                         'Savanes',   'TG', 'health_post',     3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Clinique Privée Saint-Joseph, Lomé',   '+22890001087', '33 Rue de la Paix, Lomé',                 'Maritime',  'TG', 'private',         40, true, 'TG', NOW()),
  (gen_random_uuid(), 'Clinique Privée Bon Samaritain',       '+22890001088', '18 Avenue Lomé-Nord, Lomé',               'Maritime',  'TG', 'private',         30, true, 'TG', NOW()),
  (gen_random_uuid(), 'Clinique Évangélique de Kpalimé',      '+22890001089', '7 Rue de l''Évangile, Kpalimé',           'Plateaux',  'TG', 'private',         25, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé Communautaire Kara',   '+22890001090', '9 Rue Communautaire, Kara',               'Kara',      'TG', 'community',       20, true, 'TG', NOW()),
  (gen_random_uuid(), 'Maternité de Lomé-Centrale',           '+22890001091', '24 Rue de la Maternité, Lomé',            'Maritime',  'TG', 'maternity',       40, true, 'TG', NOW()),
  (gen_random_uuid(), 'Maternité de Kara',                    '+22890001092', '11 Avenue Hospitalière, Kara',            'Kara',      'TG', 'maternity',       30, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Gbatopé',            '+22890001093', 'Gbatopé, Maritime',                       'Maritime',  'TG', 'health_post',      4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kpimé',              '+22890001094', 'Kpimé, Plateaux',                         'Plateaux',  'TG', 'health_post',      3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Lalo',               '+22890001095', 'Lalo, Plateaux',                          'Plateaux',  'TG', 'health_post',      4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Centre de Santé de Kpayando',          '+22890001096', 'Kpayando, Centrale',                      'Centrale',  'TG', 'health_center',   15, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Kparatao',           '+22890001097', 'Kparatao, Kara',                          'Kara',      'TG', 'health_post',      4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Binaparba',          '+22890001098', 'Binaparba, Savanes',                      'Savanes',   'TG', 'health_post',      3, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Toaga',              '+22890001099', 'Toaga, Savanes',                          'Savanes',   'TG', 'health_post',      4, true, 'TG', NOW()),
  (gen_random_uuid(), 'Poste de Santé de Yembour',            '+22890001100', 'Yembour, Savanes',                        'Savanes',   'TG', 'health_post',      3, true, 'TG', NOW())
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 2: kinara_patient — 1,000 test patients
-- PHI columns use SHA-256 placeholder ciphertext.
-- Does not reference clinic_id (not in V001 patients schema);
-- clinic association is managed via tenant_id at the app layer.
-- ═══════════════════════════════════════════════════════

WITH
first_names(arr) AS (SELECT ARRAY[
  'Kofi','Ama','Komi','Abla','Yao','Akosua','Kwame','Efua','Kojo','Adzo',
  'Mawuli','Sena','Dela','Yawa','Kodjo','Akua','Kwesi','Abena','Afia','Afi',
  'Atsu','Dzifa','Enam','Foli','Gbeve','Honou','Ikpidi','Judas','Keku','Lom',
  'Mina','Nubor','Ogah','Povi','Quame','Remi','Selom','Tendo','Uche','Vifah',
  'Wola','Xenia','Yvette','Zerma','Akpene','Bénédic','Céleste','Diabaté','Elom','Faustine',
  'Gaëlle','Héloïse','Iréna','Joëlle','Kékéli','Lamine','Massivi','Nadège','Olga','Pabla'
]),
last_names(arr) AS (SELECT ARRAY[
  'Adzodo','Koffi','Agbeko','Mensah','Togbedji','Dossou','Sossou','Awuitor','Badou','Amouzou',
  'Kpodo','Gblevi','Attivor','Afadjigbe','Atsou','Fiagbe','Gakpey','Hoedoafia','Klutse','Ladzekpo',
  'Novidjro','Olympio','Panka','Segbedzi','Tetteh','Voom','Woemese','Yaber','Ziope','Abalo',
  'Bessa','Creppy','Dankwa','Edoh','Gbati','Hagan','Iddi','Kudjo','Mante','Agbobli',
  'Bodjona','Chabi','Dakarai','Edjam','Fonvono','Gameli','Habia','Ikadifo','Jibril','Kazankpe',
  'Lomtchieu','Mogbante','Nakounou','Ogou','Payadou','Radimon','Sagna','Tchamdja','Ugoh','Valvo'
])
INSERT INTO patients (
    id,
    national_id_enc,
    full_name_enc,
    date_of_birth_enc,
    phone_number_enc,
    gender,
    country,
    region,
    status,
    tenant_id,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    encode(sha256(('TG-NAT-' || g.i)::bytea), 'hex'),
    encode(sha256(((f.arr)[(g.i % array_length(f.arr, 1)) + 1] || '-' ||
                   (l.arr)[((g.i * 7) % array_length(l.arr, 1)) + 1] || '-' || g.i)::bytea), 'hex'),
    encode(sha256(('TG-DOB-' || g.i)::bytea), 'hex'),
    encode(sha256(('TG-PHN-' || g.i)::bytea), 'hex'),
    CASE WHEN g.i % 2 = 0 THEN 'male' ELSE 'female' END,
    'TG',
    (ARRAY['Maritime','Plateaux','Centrale','Kara','Savanes'])[(g.i % 5) + 1],
    'active',
    'TG',
    NOW() - ((g.i % 365) || ' days')::interval,
    NOW() - ((g.i % 30) || ' days')::interval
FROM generate_series(1, 1000) AS g(i),
     first_names AS f,
     last_names AS l
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 3: kinara_farmer — 500 Togolese farmers
-- PHI columns use SHA-256 placeholder ciphertext.
-- V018 schema: full_name_enc, phone_enc (no primary_crop or currency cols).
-- ═══════════════════════════════════════════════════════
\c kinara_farmer;

WITH
farmer_names(arr) AS (SELECT ARRAY[
  'Kofi Adzodo','Ama Koffi','Komi Agbeko','Abla Mensah','Yao Togbedji',
  'Akosua Dossou','Kwame Sossou','Efua Awuitor','Kojo Badou','Adzo Amouzou',
  'Mawuli Kpodo','Sena Gblevi','Dela Attivor','Yawa Afadjigbe','Kodjo Atsou',
  'Akua Fiagbe','Kwesi Gakpey','Abena Hoedoafia','Afia Klutse','Afi Ladzekpo',
  'Atsu Novidjro','Dzifa Olympio','Enam Panka','Foli Segbedzi','Gbeve Tetteh',
  'Honou Voom','Ikpidi Woemese','Judas Yaber','Keku Ziope','Lom Abalo',
  'Mina Bessa','Nubor Creppy','Ogah Dankwa','Povi Edoh','Quame Gbati',
  'Remi Hagan','Selom Iddi','Tendo Kudjo','Uche Mante','Vifah Agbobli',
  'Wola Bodjona','Xenia Chabi','Yvette Dakarai','Zerma Edjam','Akpene Fonvono',
  'Bénédic Gameli','Céleste Habia','Diabaté Ikadifo','Elom Jibril','Faustine Kazankpe'
]),
regions(arr) AS (SELECT ARRAY['Maritime','Plateaux','Centrale','Kara','Savanes'])
INSERT INTO farmers (
    id,
    full_name_enc,
    phone_enc,
    national_id_enc,
    country,
    region,
    farm_size_ha,
    is_active,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    encode(sha256(((fn.arr)[(g.i % array_length(fn.arr, 1)) + 1] || '-' || g.i)::bytea), 'hex'),
    encode(sha256(('TG-PHN-FARMER-' || g.i)::bytea), 'hex'),
    encode(sha256((g.i || 'kinara-tg-pilot')::bytea), 'hex'),
    'TG',
    (r.arr)[(g.i % array_length(r.arr, 1)) + 1],
    round((0.5 + (g.i * 0.02 + (g.i % 10) * 0.95))::numeric, 2),
    true,
    NOW() - ((g.i % 270) || ' days')::interval,
    NOW() - ((g.i % 30) || ' days')::interval
FROM generate_series(1, 500) AS g(i),
     farmer_names AS fn,
     regions AS r
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 4: kinara_market — Initial spot price indices
-- V019 schema has price_indices (not market_prices).
-- ═══════════════════════════════════════════════════════
\c kinara_market;

INSERT INTO price_indices (id, crop_type, market, country, price_per_kg, currency, recorded_at, source) VALUES
  (gen_random_uuid(), 'maize',       'Marché de Lomé',         'TG',  275,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'maize',       'Marché de Kara',         'TG',  260,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'maize',       'Marché d''Atakpamé',     'TG',  280,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'cassava',     'Marché de Lomé',         'TG',  120,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'cassava',     'Marché de Dapaong',      'TG',  110,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'yam',         'Marché de Sokodé',       'TG',  400,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'yam',         'Marché de Kpalimé',      'TG',  380,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'cotton',      'Marché de Kara',         'TG',  350,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'coffee',      'Marché de Kpalimé',      'TG', 1800,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'cocoa',       'Marché de Kpalimé',      'TG', 2100,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'sorghum',     'Marché de Dapaong',      'TG',  220,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'millet',      'Marché de Dapaong',      'TG',  200,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'rice',        'Marché de Lomé',         'TG',  550,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'groundnuts',  'Marché de Kara',         'TG',  450,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'tomato',      'Marché de Lomé',         'TG',  300,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'maize',       'Marché de Accra',        'GH', 7800,    'GHS', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'cassava',     'Marché de Lagos',        'NG', 3500,    'NGN', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'rice',        'Marché de Nairobi',      'KE', 25000,   'KES', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'maize',       'Marché d''Abidjan',      'CI', 1200,    'XOF', CURRENT_DATE, 'togo_pilot_seed'),
  (gen_random_uuid(), 'coffee',      'Marché d''Addis-Abeba',  'ET', 95000,   'ETB', CURRENT_DATE, 'togo_pilot_seed')
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 5: kinara_payment — additional FX rate pairs
-- V040 schema has currency_rates (not fx_rates).
-- V040 already seeded USD→* pairs; this adds XOF-based and inverse pairs.
-- ═══════════════════════════════════════════════════════
\c kinara_payment;

INSERT INTO currency_rates (id, from_currency, to_currency, rate, source) VALUES
  (gen_random_uuid(), 'XOF', 'USD',  0.001700, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'EUR',  0.001524, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'GHS',  0.020800, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'NGN',  1.318000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'KES',  0.216000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'ETB',  0.087000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'TZS',  4.360000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'XOF', 'RWF',  2.128000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'GHS', 'USD',  0.081000, 'togo_pilot_seed'),
  (gen_random_uuid(), 'NGN', 'USD',  0.001300, 'togo_pilot_seed'),
  (gen_random_uuid(), 'KES', 'USD',  0.007700, 'togo_pilot_seed'),
  (gen_random_uuid(), 'ETB', 'USD',  0.009300, 'togo_pilot_seed'),
  (gen_random_uuid(), 'TZS', 'USD',  0.000390, 'togo_pilot_seed'),
  (gen_random_uuid(), 'RWF', 'USD',  0.000760, 'togo_pilot_seed'),
  (gen_random_uuid(), 'EUR', 'USD',  1.085000, 'togo_pilot_seed')
ON CONFLICT (from_currency, to_currency) DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 6: kinara_port — 10 major West African ports + 12 Lomé berths
-- V031 schema: ports(code, total_berths, status); berths(max_length_m, max_draft_m, max_tonnage_t)
-- ═══════════════════════════════════════════════════════
\c kinara_port;

INSERT INTO ports (id, name, code, country, city, latitude, longitude, total_berths, status, created_at) VALUES
  (gen_random_uuid(), 'Port Autonome de Lomé',         'TGLFW', 'TG', 'Lomé',          6.1319,   1.2730,  12, 'operational', NOW()),
  (gen_random_uuid(), 'Port of Tema',                  'GHTEM', 'GH', 'Tema',           5.6391,  -0.0064, 20, 'operational', NOW()),
  (gen_random_uuid(), 'Lagos Port Complex',            'NGLOS', 'NG', 'Lagos',          6.4541,   3.3947,  35, 'operational', NOW()),
  (gen_random_uuid(), 'Port d''Abidjan',               'CIABJ', 'CI', 'Abidjan',        5.2892,  -4.0035, 25, 'operational', NOW()),
  (gen_random_uuid(), 'Port de Cotonou',               'BJCOO', 'BJ', 'Cotonou',        6.3654,   2.4166,  10, 'operational', NOW()),
  (gen_random_uuid(), 'Port de Dakar',                 'SNDKR', 'SN', 'Dakar',         14.6928, -17.4467, 22, 'operational', NOW()),
  (gen_random_uuid(), 'Douala Port Authority',         'CMDLA', 'CM', 'Douala',          4.0511,   9.7679, 18, 'operational', NOW()),
  (gen_random_uuid(), 'Port of Mombasa',               'KEMBA', 'KE', 'Mombasa',       -4.0435,  39.6682, 30, 'operational', NOW()),
  (gen_random_uuid(), 'Port of Dar es Salaam',         'TZDAR', 'TZ', 'Dar es Salaam', -6.8199,  39.2924, 16, 'operational', NOW()),
  (gen_random_uuid(), 'Port de San-Pédro',             'CISPY', 'CI', 'San-Pédro',      4.7483,  -6.6306,  8, 'operational', NOW())
ON CONFLICT DO NOTHING;

INSERT INTO berths (id, port_id, berth_number, status, max_length_m, max_draft_m, max_tonnage_t, created_at)
SELECT
    gen_random_uuid(),
    (SELECT id FROM ports WHERE code = 'TGLFW'),
    'B' || b.n,
    CASE WHEN b.n <= 8 THEN 'available' ELSE 'occupied' END,
    (ARRAY[150,200,180,160,190,150,200,180,160,190,150,200])[b.n],
    (ARRAY[10.5,12.0,11.0,9.5,13.0,10.5,12.0,11.0,9.5,13.0,10.5,12.0])[b.n],
    (ARRAY[40000,60000,80000,50000,70000,40000,60000,80000,50000,70000,40000,60000])[b.n],
    NOW()
FROM generate_series(1, 12) AS b(n)
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 7: kinara_cooperative — 10 pilot cooperatives
-- V020 schema: registration_no (not registration_number), total_members (not member_count).
-- ═══════════════════════════════════════════════════════
\c kinara_cooperative;

INSERT INTO cooperatives (id, name, registration_no, country, region, total_members, contact_phone, description, status, created_at) VALUES
  (gen_random_uuid(), 'Coopérative Maïs du Maritime',        'COOP-TG-001', 'TG', 'Maritime', 120, '+22890010001', 'maize production cooperative',   'active', NOW()),
  (gen_random_uuid(), 'Union des Caféiculteurs de Kpalimé',  'COOP-TG-002', 'TG', 'Plateaux',  85, '+22890010002', 'coffee growers union',           'active', NOW()),
  (gen_random_uuid(), 'Coopérative Coton du Nord',           'COOP-TG-003', 'TG', 'Kara',     200, '+22890010003', 'cotton production cooperative',   'active', NOW()),
  (gen_random_uuid(), 'Association Cacaoyère de Badou',      'COOP-TG-004', 'TG', 'Plateaux',  65, '+22890010004', 'cocoa growers association',       'active', NOW()),
  (gen_random_uuid(), 'Groupement Vivrier de la Centrale',   'COOP-TG-005', 'TG', 'Centrale', 150, '+22890010005', 'yam and food crop collective',    'active', NOW()),
  (gen_random_uuid(), 'Coopérative Sorgho des Savanes',      'COOP-TG-006', 'TG', 'Savanes',  175, '+22890010006', 'sorghum production cooperative',  'active', NOW()),
  (gen_random_uuid(), 'Union Maraîchère de Lomé',            'COOP-TG-007', 'TG', 'Maritime',  90, '+22890010007', 'vegetable and tomato growers',    'active', NOW()),
  (gen_random_uuid(), 'Coopérative Riz du Lac Togo',         'COOP-TG-008', 'TG', 'Maritime',  60, '+22890010008', 'rice production cooperative',     'active', NOW()),
  (gen_random_uuid(), 'Association Femmes Agricultrices',    'COOP-TG-009', 'TG', 'Maritime', 250, '+22890010009', 'groundnut and women''s farming',  'active', NOW()),
  (gen_random_uuid(), 'Coopérative Mil et Sorgho Nord-Togo', 'COOP-TG-010', 'TG', 'Savanes',  130, '+22890010010', 'millet and sorghum collective',   'active', NOW())
ON CONFLICT (registration_no) DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 8: kinara_notification — notification templates
-- V044 schema: template_key (UNIQUE), body_template (not body).
-- Keys include language suffix to satisfy UNIQUE constraint.
-- ═══════════════════════════════════════════════════════
\c kinara_notification;

INSERT INTO notification_templates (id, template_key, channel, language, subject, body_template, is_active, created_at) VALUES
  (gen_random_uuid(), 'appointment_reminder_fr',  'sms', 'fr', NULL,
   'KINARA RDV: Rappel - Votre consultation est demain à {{time}} à {{clinic}}. Ref: {{ref}}',
   true, NOW()),
  (gen_random_uuid(), 'appointment_reminder_en',  'sms', 'en', NULL,
   'KINARA APPT: Reminder - Your appointment is tomorrow at {{time}} at {{clinic}}. Ref: {{ref}}',
   true, NOW()),
  (gen_random_uuid(), 'lab_result_ready_fr',      'sms', 'fr', NULL,
   'KINARA LABO: Vos résultats pour {{test}} sont prêts. Ordre: {{ref}}. Collectez au labo.',
   true, NOW()),
  (gen_random_uuid(), 'outbreak_alert_fr',        'sms', 'fr', NULL,
   'ALERTE SANTE KINARA: {{disease}} signalée dans votre région ({{region}}). Cas: {{count}}. Consultez le CS local.',
   true, NOW()),
  (gen_random_uuid(), 'payment_received_fr',      'sms', 'fr', NULL,
   'KINARA PAY: {{amount}} {{currency}} reçu de {{sender}}. Solde: {{balance}} {{currency}}.',
   true, NOW()),
  (gen_random_uuid(), 'market_price_alert_fr',    'sms', 'fr', NULL,
   'KINARA PRIX: {{commodity}} à {{price}} {{currency}}/kg au marché de {{market}}.',
   true, NOW())
ON CONFLICT (template_key) DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- Verification queries (run manually after migration)
-- ═══════════════════════════════════════════════════════
-- \c kinara_patient;      SELECT 'clinics'  , COUNT(*) FROM clinics;
-- \c kinara_patient;      SELECT 'patients' , COUNT(*) FROM patients;
-- \c kinara_farmer;       SELECT 'farmers'  , COUNT(*) FROM farmers;
-- \c kinara_market;       SELECT 'prices'   , COUNT(*) FROM price_indices WHERE source = 'togo_pilot_seed';
-- \c kinara_payment;      SELECT 'fx_rates' , COUNT(*) FROM currency_rates;
-- \c kinara_port;         SELECT 'ports'    , COUNT(*) FROM ports;
-- \c kinara_port;         SELECT 'berths'   , COUNT(*) FROM berths;
-- \c kinara_cooperative;  SELECT 'coops'    , COUNT(*) FROM cooperatives;
-- \c kinara_notification; SELECT 'templates', COUNT(*) FROM notification_templates;

-- DOWN: undo seed (dev reset only — never run in production without sign-off)
-- \c kinara_patient;      TRUNCATE clinics, patients CASCADE;
-- \c kinara_farmer;       TRUNCATE farmers CASCADE;
-- \c kinara_market;       DELETE FROM price_indices WHERE source = 'togo_pilot_seed';
-- \c kinara_payment;      DELETE FROM currency_rates WHERE source = 'togo_pilot_seed';
-- \c kinara_port;         TRUNCATE ports, berths CASCADE;
-- \c kinara_cooperative;  DELETE FROM cooperatives WHERE registration_no LIKE 'COOP-TG-%';
-- \c kinara_notification; DELETE FROM notification_templates WHERE template_key LIKE 'appointment_%' OR template_key LIKE 'lab_%' OR template_key LIKE 'outbreak_%' OR template_key LIKE 'payment_%' OR template_key LIKE 'market_%';
