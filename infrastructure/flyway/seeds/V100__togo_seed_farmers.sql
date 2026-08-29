\c kinara_farmer;

-- 100 test farmers in Togo
INSERT INTO farmers (id, name, phone, national_id_enc, country, region, primary_crop, farm_size_ha, currency, is_active, created_at, updated_at)
SELECT
    gen_random_uuid(),
    names.name,
    '+228' || (90000000 + row_number() OVER())::text,
    encode(gen_random_bytes(16), 'hex'),
    'TG',
    regions.region,
    crops.crop,
    (random() * 10 + 0.5)::numeric(5,2),
    'XOF',
    true,
    NOW() - (random() * interval '90 days'),
    NOW()
FROM
    (VALUES
        ('Kofi Adzodo'), ('Ama Koffi'), ('Komi Agbeko'), ('Abla Mensah'), ('Yao Togbedji'),
        ('Akosua Dossou'), ('Kwame Sossou'), ('Efua Awuitor'), ('Kojo Badou'), ('Adzo Amouzou'),
        ('Mawuli Kpodo'), ('Sena Gblevi'), ('Dela Attivor'), ('Yawa Afadjigbe'), ('Kodjo Atsou'),
        ('Akua Dossou'), ('Kwesi Fiagbe'), ('Abena Gakpey'), ('Komi Hoedoafia'), ('Afia Klutse'),
        ('Yao Ladzekpo'), ('Afi Mensah'), ('Koffi Novidjro'), ('Abla Olympio'), ('Komivi Panka'),
        ('Ama Quist'), ('Kofi Reddy'), ('Akua Segbedzi'), ('Kojo Tetteh'), ('Efua Usman'),
        ('Mawuli Voom'), ('Sena Woemese'), ('Dela Xochitl'), ('Yawa Yaber'), ('Kodjo Ziope'),
        ('Akosua Abalo'), ('Kwame Bessa'), ('Efua Creppy'), ('Kojo Dankwa'), ('Adzo Edoh'),
        ('Kofi Fabio'), ('Ama Gbati'), ('Komi Hagan'), ('Abla Iddi'), ('Yao Johnson'),
        ('Akua Kudjo'), ('Kwesi Lom'), ('Abena Mante'), ('Komi Nuviadenu'), ('Afia Oti'),
        ('Yao Pivi'), ('Afi Quarcoo'), ('Koffi Regent'), ('Abla Seto'), ('Komivi Tchamdja'),
        ('Ama Ucha'), ('Kofi Vigninou'), ('Akua Weto'), ('Kojo Xoese'), ('Efua Yawo'),
        ('Mawuli Zorgo'), ('Sena Agbenyega'), ('Dela Boateng'), ('Yawa Chieku'), ('Kodjo Dotse'),
        ('Akosua Ekpe'), ('Kwame Foli'), ('Efua Gakpo'), ('Kojo Hevi'), ('Adzo Issaka'),
        ('Kofi James'), ('Ama Kodzo'), ('Komi Lawson'), ('Abla Mantey'), ('Yao Negble'),
        ('Akua Ofori'), ('Kwesi Puplampu'), ('Abena Quaye'), ('Komi Rossiter'), ('Afia Segla'),
        ('Yao Tay'), ('Afi Ugbator'), ('Koffi Vordoagu'), ('Abla Wakle'), ('Komivi Xotor'),
        ('Ama Yade'), ('Kofi Zottor'), ('Akua Amewunou'), ('Kojo Botwe'), ('Efua Cobblah'),
        ('Mawuli Dikeni'), ('Sena Edorh'), ('Dela Fianu'), ('Yawa Gbadago'), ('Kodjo Hotor'),
        ('Akosua Ido'), ('Kwame Jeannot'), ('Efua Klika'), ('Kojo Lamptey'), ('Adzo Mawuko')
    ) AS names(name),
    (VALUES ('Maritime'), ('Plateaux'), ('Centrale'), ('Kara'), ('Savanes'),
            ('Maritime'), ('Plateaux'), ('Centrale'), ('Kara'), ('Savanes')) AS regions(region),
    (VALUES ('maize'), ('cassava'), ('yam'), ('sorghum'), ('millet'),
            ('cotton'), ('soybean'), ('groundnut'), ('rice'), ('cowpea')) AS crops(crop)
LIMIT 100;
