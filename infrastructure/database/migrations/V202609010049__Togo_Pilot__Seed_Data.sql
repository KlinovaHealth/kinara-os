-- V049: Togo Pilot Seed Data
-- Seeds 100 clinics, 1000 patients, 500 farmers, 10 ports, FX rates,
-- 5 cooperatives, and initial market prices for the Oct 2026 Togo pilot.
-- All inserts are idempotent (ON CONFLICT DO NOTHING).

-- ═══════════════════════════════════════════════════════
-- SECTION 1: kinara_patient — Clinics (100 Togolese sites)
-- ═══════════════════════════════════════════════════════
\c kinara_patient;

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
  -- Clinics 51–100: community health posts across all regions
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
-- Uses generate_series for efficiency; distributed across 100 clinics.
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
]),
clinic_ids AS (
    SELECT id, row_number() OVER (ORDER BY created_at) AS rn FROM clinics WHERE country = 'TG'
)
INSERT INTO patients (
    id, patient_ref, first_name, last_name, date_of_birth, gender,
    blood_type, phone, country, tenant_id, clinic_id, is_active, created_at
)
SELECT
    gen_random_uuid(),
    'PAT-' || upper(lpad(to_hex(g.i), 8, '0')),
    (f.arr)[(g.i % array_length(f.arr, 1)) + 1],
    (l.arr)[((g.i * 7) % array_length(l.arr, 1)) + 1],
    CURRENT_DATE - ((15 + (g.i * 97) % 7300) || ' days')::interval,
    CASE WHEN g.i % 2 = 0 THEN 'male' ELSE 'female' END,
    (ARRAY['A+','A-','B+','B-','AB+','AB-','O+','O-'])[(g.i % 8) + 1],
    '+228' || lpad(((90000000 + g.i * 12347) % 89999999 + 10000000)::text, 8, '0'),
    'TG',
    'TG',
    (SELECT id FROM clinic_ids WHERE rn = (g.i % 100) + 1),
    true,
    NOW() - ((g.i % 365) || ' days')::interval
FROM generate_series(1, 1000) AS g(i),
     first_names AS f,
     last_names AS l
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 3: kinara_farmer — 500 Togolese farmers
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
regions(arr) AS (SELECT ARRAY['Maritime','Plateaux','Centrale','Kara','Savanes']),
crops(arr) AS (SELECT ARRAY[
  'maize','cassava','yam','cotton','coffee','cocoa','sorghum','millet',
  'rice','beans','groundnuts','sesame','tomato','pepper','plantain'
])
INSERT INTO farmers (
    id, name, phone, national_id_enc, country, region,
    primary_crop, farm_size_ha, currency, is_active, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    (fn.arr)[(g.i % array_length(fn.arr, 1)) + 1] || ' #' || g.i,
    '+228' || lpad(((90000000 + g.i * 23456) % 89999999 + 10000000)::text, 8, '0'),
    encode(sha256((g.i || 'kinara-tg-pilot')::bytea), 'hex'),
    'TG',
    (r.arr)[(g.i % array_length(r.arr, 1)) + 1],
    (c.arr)[(g.i % array_length(c.arr, 1)) + 1],
    round((0.5 + (g.i * 0.02 + random() * 9.5))::numeric, 2),
    'XOF',
    true,
    NOW() - ((g.i % 270) || ' days')::interval,
    NOW() - ((g.i % 30) || ' days')::interval
FROM generate_series(1, 500) AS g(i),
     farmer_names AS fn,
     regions AS r,
     crops AS c
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 4: kinara_market — Initial market prices
-- ═══════════════════════════════════════════════════════
\c kinara_market;

INSERT INTO market_prices (id, commodity_name, price_per_unit, unit, market_name, country, currency, recorded_at) VALUES
  (gen_random_uuid(), 'maize',       275, 'kg', 'Marché de Lomé',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'maize',       260, 'kg', 'Marché de Kara',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'maize',       280, 'kg', 'Marché d''Atakpamé',     'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'cassava',     120, 'kg', 'Marché de Lomé',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'cassava',     110, 'kg', 'Marché de Dapaong',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'yam',         400, 'kg', 'Marché de Sokodé',       'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'yam',         380, 'kg', 'Marché de Kpalimé',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'cotton',      350, 'kg', 'Marché de Kara',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'coffee',     1800, 'kg', 'Marché de Kpalimé',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'cocoa',      2100, 'kg', 'Marché de Kpalimé',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'sorghum',     220, 'kg', 'Marché de Dapaong',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'millet',      200, 'kg', 'Marché de Dapaong',      'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'rice',        550, 'kg', 'Marché de Lomé',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'groundnuts',  450, 'kg', 'Marché de Kara',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'tomato',      300, 'kg', 'Marché de Lomé',         'TG', 'XOF', NOW()),
  (gen_random_uuid(), 'maize',      7800, 'kg', 'Marché de Accra',        'GH', 'GHS', NOW()),
  (gen_random_uuid(), 'cassava',    3500, 'kg', 'Marché de Lagos',        'NG', 'NGN', NOW()),
  (gen_random_uuid(), 'rice',      25000, 'kg', 'Marché de Nairobi',      'KE', 'KES', NOW()),
  (gen_random_uuid(), 'maize',      1200, 'kg', 'Marché d''Abidjan',      'CI', 'XOF', NOW()),
  (gen_random_uuid(), 'coffee',    95000, 'kg', 'Marché d''Addis-Abeba',  'ET', 'ETB', NOW())
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 5: kinara_payment — FX rates (Togo pilot currencies)
-- ═══════════════════════════════════════════════════════
\c kinara_payment;

INSERT INTO fx_rates (id, from_currency, to_currency, rate, source, updated_at) VALUES
  (gen_random_uuid(), 'XOF', 'USD',  0.001700, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'EUR',  0.001524, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'GHS',  0.020800, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'NGN',  1.318000, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'KES',  0.216000, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'ETB',  0.087000, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'TZS',  4.360000, 'seed', NOW()),
  (gen_random_uuid(), 'XOF', 'RWF',  2.128000, 'seed', NOW()),
  (gen_random_uuid(), 'GHS', 'USD',  0.081000, 'seed', NOW()),
  (gen_random_uuid(), 'NGN', 'USD',  0.001300, 'seed', NOW()),
  (gen_random_uuid(), 'KES', 'USD',  0.007700, 'seed', NOW()),
  (gen_random_uuid(), 'ETB', 'USD',  0.009300, 'seed', NOW()),
  (gen_random_uuid(), 'TZS', 'USD',  0.000390, 'seed', NOW()),
  (gen_random_uuid(), 'RWF', 'USD',  0.000760, 'seed', NOW()),
  (gen_random_uuid(), 'EUR', 'USD',  1.085000, 'seed', NOW()),
  (gen_random_uuid(), 'USD', 'XOF', 588.200000,'seed', NOW()),
  (gen_random_uuid(), 'USD', 'GHS',  12.35000, 'seed', NOW()),
  (gen_random_uuid(), 'USD', 'NGN', 769.00000, 'seed', NOW()),
  (gen_random_uuid(), 'USD', 'KES', 129.87000, 'seed', NOW()),
  (gen_random_uuid(), 'USD', 'ETB', 107.53000, 'seed', NOW())
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 6: kinara_port — 10 major West African ports
-- ═══════════════════════════════════════════════════════
\c kinara_port;

INSERT INTO ports (id, name, port_code, country, city, latitude, longitude, berths_total, berths_available, max_vessel_dwt, timezone, is_active, created_at) VALUES
  (gen_random_uuid(), 'Port Autonome de Lomé',         'TGLFW', 'TG', 'Lomé',          6.1319, 1.2730,   12, 8,  80000, 'Africa/Abidjan', true, NOW()),
  (gen_random_uuid(), 'Port of Tema',                  'GHTEM', 'GH', 'Tema',           5.6391, -0.0064,  20,14, 100000, 'Africa/Accra',   true, NOW()),
  (gen_random_uuid(), 'Lagos Port Complex',            'NGLOS', 'NG', 'Lagos',          6.4541, 3.3947,   35,20, 150000, 'Africa/Lagos',   true, NOW()),
  (gen_random_uuid(), 'Port d''Abidjan',               'CIABJ', 'CI', 'Abidjan',        5.2892, -4.0035,  25,18, 120000, 'Africa/Abidjan', true, NOW()),
  (gen_random_uuid(), 'Port de Cotonou',               'BJCOO', 'BJ', 'Cotonou',        6.3654, 2.4166,   10, 7,  60000, 'Africa/Porto-Novo',true,NOW()),
  (gen_random_uuid(), 'Port de Dakar',                 'SNDKR', 'SN', 'Dakar',          14.6928,-17.4467, 22,15, 110000, 'Africa/Dakar',   true, NOW()),
  (gen_random_uuid(), 'Douala Port Authority',         'CMDLA', 'CM', 'Douala',          4.0511, 9.7679,   18,12,  90000, 'Africa/Douala',  true, NOW()),
  (gen_random_uuid(), 'Port of Mombasa',               'KEMBA', 'KE', 'Mombasa',       -4.0435,39.6682,   30,20, 130000, 'Africa/Nairobi', true, NOW()),
  (gen_random_uuid(), 'Port of Dar es Salaam',         'TZDAR', 'TZ', 'Dar es Salaam', -6.8199,39.2924,   16,11,  85000, 'Africa/Dar_es_Salaam',true,NOW()),
  (gen_random_uuid(), 'Port de San-Pédro',             'CISPY', 'CI', 'San-Pédro',      4.7483,-6.6306,    8, 6,  60000, 'Africa/Abidjan', true, NOW())
ON CONFLICT DO NOTHING;

-- Seed berths for Lomé port
INSERT INTO berths (id, port_id, berth_number, berth_type, max_dwt_tonnes, length_m, depth_m, status, created_at)
SELECT
    gen_random_uuid(),
    (SELECT id FROM ports WHERE port_code = 'TGLFW'),
    'B' || b.n,
    (ARRAY['general_cargo','container','bulk','tanker','roro'])[(b.n % 5) + 1],
    (ARRAY[40000,60000,80000,50000,70000])[(b.n % 5) + 1],
    (ARRAY[150,200,180,160,190])[(b.n % 5) + 1],
    (ARRAY[10.5,12.0,11.0,9.5,13.0])[(b.n % 5) + 1],
    CASE WHEN b.n <= 8 THEN 'available' ELSE 'occupied' END,
    NOW()
FROM generate_series(1, 12) AS b(n)
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 7: kinara_cooperative — 10 pilot cooperatives
-- ═══════════════════════════════════════════════════════
\c kinara_cooperative;

INSERT INTO cooperatives (id, name, region, country, crop_focus, member_count, registration_number, contact_phone, is_active, created_at) VALUES
  (gen_random_uuid(), 'Coopérative Maïs du Maritime',        'Maritime', 'TG', 'maize',      120, 'COOP-TG-001', '+22890010001', true, NOW()),
  (gen_random_uuid(), 'Union des Caféiculteurs de Kpalimé',  'Plateaux', 'TG', 'coffee',      85, 'COOP-TG-002', '+22890010002', true, NOW()),
  (gen_random_uuid(), 'Coopérative Coton du Nord',           'Kara',     'TG', 'cotton',     200, 'COOP-TG-003', '+22890010003', true, NOW()),
  (gen_random_uuid(), 'Association Cacaoyère de Badou',      'Plateaux', 'TG', 'cocoa',       65, 'COOP-TG-004', '+22890010004', true, NOW()),
  (gen_random_uuid(), 'Groupement Vivrier de la Centrale',   'Centrale', 'TG', 'yam',        150, 'COOP-TG-005', '+22890010005', true, NOW()),
  (gen_random_uuid(), 'Coopérative Sorgho des Savanes',      'Savanes',  'TG', 'sorghum',    175, 'COOP-TG-006', '+22890010006', true, NOW()),
  (gen_random_uuid(), 'Union Maraîchère de Lomé',            'Maritime', 'TG', 'tomato',      90, 'COOP-TG-007', '+22890010007', true, NOW()),
  (gen_random_uuid(), 'Coopérative Riz du Lac Togo',         'Maritime', 'TG', 'rice',        60, 'COOP-TG-008', '+22890010008', true, NOW()),
  (gen_random_uuid(), 'Association Femmes Agricultrices',    'Maritime', 'TG', 'groundnuts', 250, 'COOP-TG-009', '+22890010009', true, NOW()),
  (gen_random_uuid(), 'Coopérative Mil et Sorgho Nord-Togo', 'Savanes',  'TG', 'millet',     130, 'COOP-TG-010', '+22890010010', true, NOW())
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- SECTION 8: kinara_notification — notification templates
-- ═══════════════════════════════════════════════════════
\c kinara_notification;

INSERT INTO notification_templates (id, name, channel, language, subject, body, is_active, created_at) VALUES
  (gen_random_uuid(), 'appointment_reminder',    'sms', 'fr', NULL,
   'KINARA RDV: Rappel - Votre consultation est demain à {{time}} à {{clinic}}. Ref: {{ref}}',
   true, NOW()),
  (gen_random_uuid(), 'appointment_reminder',    'sms', 'en', NULL,
   'KINARA APPT: Reminder - Your appointment is tomorrow at {{time}} at {{clinic}}. Ref: {{ref}}',
   true, NOW()),
  (gen_random_uuid(), 'lab_result_ready',        'sms', 'fr', NULL,
   'KINARA LABO: Vos résultats pour {{test}} sont prêts. Ordre: {{ref}}. Collectez au labo.',
   true, NOW()),
  (gen_random_uuid(), 'outbreak_alert',          'sms', 'fr', NULL,
   'ALERTE SANTE KINARA: {{disease}} signalée dans votre région ({{region}}). Cas: {{count}}. Consultez le CS local.',
   true, NOW()),
  (gen_random_uuid(), 'payment_received',        'sms', 'fr', NULL,
   'KINARA PAY: {{amount}} {{currency}} reçu de {{sender}}. Solde: {{balance}} {{currency}}.',
   true, NOW()),
  (gen_random_uuid(), 'market_price_alert',      'sms', 'fr', NULL,
   'KINARA PRIX: {{commodity}} à {{price}} {{currency}}/kg au marché de {{market}}.',
   true, NOW())
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════
-- Verification queries (run manually after migration)
-- ═══════════════════════════════════════════════════════
-- \c kinara_patient;  SELECT 'clinics'  , COUNT(*) FROM clinics;
-- \c kinara_patient;  SELECT 'patients' , COUNT(*) FROM patients;
-- \c kinara_farmer;   SELECT 'farmers'  , COUNT(*) FROM farmers;
-- \c kinara_market;   SELECT 'prices'   , COUNT(*) FROM market_prices;
-- \c kinara_payment;  SELECT 'fx_rates' , COUNT(*) FROM fx_rates;
-- \c kinara_port;     SELECT 'ports'    , COUNT(*) FROM ports;
-- \c kinara_cooperative; SELECT 'coops' , COUNT(*) FROM cooperatives;

-- DOWN: truncate all seeded tables (dangerous — only for dev reset)
-- \c kinara_patient;   TRUNCATE clinics, patients CASCADE;
-- \c kinara_farmer;    TRUNCATE farmers CASCADE;
-- \c kinara_market;    TRUNCATE market_prices CASCADE;
-- \c kinara_payment;   DELETE FROM fx_rates WHERE source = 'seed';
-- \c kinara_port;      TRUNCATE ports, berths CASCADE;
-- \c kinara_cooperative; TRUNCATE cooperatives CASCADE;
