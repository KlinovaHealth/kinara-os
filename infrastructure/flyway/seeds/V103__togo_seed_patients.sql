-- 100 test patients in Togo
\c kinara_patient;

INSERT INTO patients (id, patient_ref, first_name, last_name, date_of_birth, gender, phone_enc, national_id_enc, country, region, blood_type, is_active, tenant_id, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'PAT-' || upper(substr(gen_random_uuid()::text, 1, 8)),
    first_names.fn,
    last_names.ln,
    (NOW() - (random() * interval '60 years' + interval '1 year'))::date,
    (ARRAY['M','F'])[floor(random()*2+1)],
    encode(gen_random_bytes(32), 'base64'),   -- phone_enc (AES-256-GCM placeholder)
    encode(gen_random_bytes(32), 'base64'),   -- national_id_enc
    'TG',
    (ARRAY['Maritime','Plateaux','Centrale','Kara','Savanes'])[floor(random()*5+1)],
    (ARRAY['A+','A-','B+','B-','AB+','AB-','O+','O-'])[floor(random()*8+1)],
    true,
    'TG',
    NOW() - (random() * interval '180 days'),
    NOW()
FROM
    (VALUES
        ('Kofi'), ('Ama'), ('Komi'), ('Abla'), ('Yao'), ('Akosua'), ('Kwame'), ('Efua'),
        ('Kojo'), ('Adzo'), ('Mawuli'), ('Sena'), ('Dela'), ('Yawa'), ('Kodjo'), ('Akua'),
        ('Kwesi'), ('Abena'), ('Afia'), ('Afi'), ('Komivi'), ('Adzovi'), ('Kafui'), ('Selom'),
        ('Edem'), ('Dzifa'), ('Senanu'), ('Mawuena'), ('Togbe'), ('Dzigbodi'),
        ('Gideon'), ('Patience'), ('Emmanuel'), ('Grace'), ('Daniel'), ('Abigail'),
        ('Samuel'), ('Miriam'), ('Joseph'), ('Comfort'), ('Isaac'), ('Rejoice'),
        ('Moses'), ('Felicity'), ('Aaron'), ('Dorcas'), ('Caleb'), ('Blessing'),
        ('Solomon'), ('Priscilla'), ('David'), ('Rebecca'), ('Joshua'), ('Esther'),
        ('Elijah'), ('Ruth'), ('Jeremiah'), ('Lydia'), ('Ezekiel'), ('Naomi'),
        ('Hosea'), ('Sarah'), ('Joel'), ('Deborah'), ('Amos'), ('Hannah'),
        ('Obadiah'), ('Leah'), ('Jonah'), ('Rachel'), ('Micah'), ('Miriam'),
        ('Nahum'), ('Abigail'), ('Habakkuk'), ('Bathsheba'), ('Zephaniah'), ('Tamar'),
        ('Haggai'), ('Dinah'), ('Zechariah'), ('Zilpah'), ('Malachi'), ('Bilhah'),
        ('Aziz'), ('Fatima'), ('Ibrahim'), ('Aminata'), ('Moussa'), ('Mariam'),
        ('Soulé'), ('Houéfa'), ('Idrissou'), ('Ramatou'), ('Abdou'), ('Adjoa'),
        ('Hamidou'), ('Nathalie'), ('Oumar'), ('Christine')
    ) AS first_names(fn),
    (VALUES
        ('Adzodo'), ('Koffi'), ('Agbeko'), ('Mensah'), ('Togbedji'), ('Dossou'),
        ('Sossou'), ('Awuitor'), ('Badou'), ('Amouzou'), ('Kpodo'), ('Gblevi'),
        ('Attivor'), ('Afadjigbe'), ('Atsou'), ('Fiagbe'), ('Gakpey'), ('Hoedoafia'),
        ('Klutse'), ('Ladzekpo'), ('Novidjro'), ('Olympio'), ('Panka'), ('Segbedzi'),
        ('Abalo'), ('Bessa'), ('Creppy'), ('Dankwa'), ('Edoh'), ('Gbati')
    ) AS last_names(ln)
LIMIT 100
ON CONFLICT DO NOTHING;
