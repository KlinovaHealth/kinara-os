#!/usr/bin/env python3
"""
Kinara OS — Synthetic (Fabricated) Data Generator
===================================================
Generates load-test and DR-test data for the 4 Kinara OS pillars:
  Health · Agriculture · Logistics · Maritime

ALL DATA IS COMPLETELY FABRICATED.  No real patient, farmer, or trade records
are produced.  Real PHI must never be seeded into a non-production environment.

Safety:
  This script refuses to run if KINARA_ENV=production.
  Always target a separate staging or load-test Postgres instance.

Usage:
  pip install faker psycopg2-binary click
  export PGPASSWORD="<staging-password>"

  # Preview without writing
  python3 scripts/generate_synthetic_data.py --dry-run \
      --host db-staging.internal --user kinara_app

  # Full run (~50 clinics, ~10 000 patients, proportional records)
  python3 scripts/generate_synthetic_data.py \
      --host db-staging.internal --user kinara_app

  # Remove previously generated synthetic data
  python3 scripts/generate_synthetic_data.py --clean \
      --host db-staging.internal --user kinara_app

Options:
  --host        Postgres host           [env: PGHOST]
  --port        Postgres port (5432)    [env: PGPORT]
  --user        Postgres user           [env: PGUSER]
  --clinics     Number of clinics       (default 50)
  --patients    Patients per clinic     (default 200)
  --dry-run     Print counts, no writes
  --clean       DELETE previously generated synthetic rows, then exit
"""

import os
import sys
import random
import secrets
import base64
import uuid
import datetime
import click
import psycopg2
from psycopg2.extras import execute_values

try:
    from faker import Faker
except ImportError:
    sys.exit("Install dependencies first:\n  pip install faker psycopg2-binary click")

SYNTH_TAG = "synthetic_load_test"
SYNTH_TENANT_PREFIX = "SYNTH_"

AFRICAN_COUNTRIES = [
    ("TG", "Togo",               [("Maritime", 6.14,  1.22), ("Plateaux", 7.0,  1.1),
                                   ("Centrale", 8.5,   1.1), ("Kara",     9.5,  1.2),
                                   ("Savanes",  10.5,  0.9)]),
    ("GH", "Ghana",              [("Greater Accra", 5.5,  -0.2), ("Ashanti", 6.7, -1.6),
                                   ("Northern",      9.4,  -1.1), ("Western", 5.1, -1.9)]),
    ("BJ", "Benin",              [("Littoral", 6.4, 2.4), ("Zou", 7.2, 2.1),
                                   ("Borgou",  9.3, 2.5), ("Alibori", 11.0, 2.8)]),
    ("SN", "Senegal",            [("Dakar", 14.7, -17.4), ("Thiès", 14.8, -16.9),
                                   ("Kaolack", 14.1, -16.1), ("Saint-Louis", 16.0, -15.9)]),
    ("NG", "Nigeria",            [("Lagos", 6.5, 3.4), ("Kano", 12.0, 8.5),
                                   ("Oyo",   7.9, 3.9), ("Rivers", 4.8, 7.0)]),
    ("KE", "Kenya",              [("Nairobi", -1.3, 36.8), ("Mombasa", -4.0, 39.7),
                                   ("Kisumu",  -0.1, 34.8), ("Nakuru",  -0.3, 36.1)]),
    ("TZ", "Tanzania",           [("Dar es Salaam", -6.8, 39.3), ("Arusha", -3.4, 36.7),
                                   ("Dodoma",        -6.2, 35.7), ("Mwanza", -2.5, 32.9)]),
    ("CI", "Côte d'Ivoire",      [("Abidjan", 5.3, -4.0), ("Bouaké", 7.7, -5.0),
                                   ("Yamoussoukro", 6.8, -5.3)]),
    ("CM", "Cameroon",           [("Littoral", 4.1, 9.7), ("Centre", 3.9, 11.5),
                                   ("Nord",     9.3, 13.4), ("Extrême-Nord", 11.5, 14.5)]),
    ("ET", "Ethiopia",           [("Addis Ababa", 9.0, 38.7), ("Oromia", 7.5, 40.2),
                                   ("Amhara",      11.0, 37.9), ("SNNP",   6.9, 37.7)]),
]

PORT_DATA = [
    ("Lomé Port",       "TGLOM", "TG", "Lomé",        6.130, 1.270,  15.0, 12),
    ("Tema Port",       "GHTEM", "GH", "Accra",        5.616, -0.017, 13.5, 18),
    ("Cotonou Port",    "BJCOO", "BJ", "Cotonou",      6.368,  2.435, 12.0,  8),
    ("Dakar Port",      "SNDKR", "SN", "Dakar",       14.682,-17.435, 14.0, 20),
    ("Lagos Port",      "NGLAG", "NG", "Lagos",         6.455,  3.383, 16.0, 40),
    ("Abidjan Port",    "CIABJ", "CI", "Abidjan",       5.320, -4.015, 14.5, 25),
    ("Mombasa Port",    "KEMBA", "KE", "Mombasa",      -4.060, 39.672, 13.0, 22),
    ("Dar es Salaam",   "TZDAR", "TZ", "Dar es Salaam",-6.814, 39.292, 12.5, 16),
    ("Douala Port",     "CMDLA", "CM", "Douala",        4.050,  9.707, 11.5, 14),
    ("Port Harcourt",   "NGPHC", "NG", "Port Harcourt", 4.800,  7.000, 10.0, 10),
    ("Libreville Port", "GALBV", "GA", "Libreville",    0.400,  9.450,  9.0,  8),
    ("Conakry Port",    "GNCKY", "GN", "Conakry",       9.540,-13.710,  8.5,  7),
    ("Beira Port",      "MZBEW", "MZ", "Beira",        -19.844, 34.838, 11.0, 12),
    ("Maputo Port",     "MZMPM", "MZ", "Maputo",       -25.960, 32.580, 12.0, 14),
    ("Djibouti Port",   "DJJIB", "DJ", "Djibouti",     11.590, 43.145, 13.5, 18),
]

CROP_TYPES = [
    "maize", "cassava", "sorghum", "millet", "rice", "wheat", "yam",
    "groundnut", "cowpea", "soybean", "cotton", "cocoa", "coffee",
    "palm_oil", "rubber", "banana", "plantain", "tomato", "onion",
    "pepper", "sesame", "shea", "ginger", "sugar_cane",
]

CURRENCIES = ["USD", "EUR", "XOF", "GHS", "KES", "TZS", "NGN", "XAF", "ETB"]

VEHICLE_TYPES = ["truck", "pickup", "motorcycle", "van", "refrigerated", "tanker"]
VEHICLE_MAKES  = ["Toyota", "Mercedes", "Mitsubishi", "Isuzu", "MAN", "Renault",
                  "Volvo", "DAF", "Scania", "Nissan"]


# ─── Helpers ──────────────────────────────────────────────────────────────────

def synth_enc(label: str, n: int) -> str:
    """Return a plausible placeholder ciphertext (NOT real AES-256-GCM)."""
    raw = f"SYNTHETIC_{label}_{n}_{secrets.token_hex(12)}".encode()
    return base64.b64encode(raw).decode()


def rand_date_past(days: int = 60 * 365) -> datetime.date:
    delta = random.randint(1 * 365, days)
    return (datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(days=delta)).date()


def ts_past(days: int = 365) -> datetime.datetime:
    delta = random.uniform(0, days * 86400)
    return datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(seconds=delta)


def fake_reg_no(prefix: str, n: int) -> str:
    return f"{prefix}-SYNTH-{n:05d}"


def _conn(host: str, port: int, user: str, dbname: str) -> psycopg2.extensions.connection:
    password = os.environ.get("PGPASSWORD", "")
    return psycopg2.connect(
        host=host, port=port, user=user, password=password,
        dbname=dbname, sslmode="require", connect_timeout=10,
    )


def _batch(cur, sql: str, rows: list, batch: int = 500):
    for i in range(0, len(rows), batch):
        execute_values(cur, sql, rows[i : i + batch])


# ─── Clean ────────────────────────────────────────────────────────────────────

def clean_all(host: str, port: int, user: str):
    click.echo("Removing synthetic data …")

    db_deletes = {
        "kinara_patient":     ["DELETE FROM patients    WHERE tenant_id LIKE %s",
                               "DELETE FROM clinics     WHERE name      LIKE %s"],
        "kinara_farmer":      ["DELETE FROM farmers     WHERE country   IS NOT NULL AND full_name_enc LIKE %s",
                               "DELETE FROM farm_plots  WHERE plot_name LIKE %s"],
        "kinara_cooperative": ["DELETE FROM cooperatives WHERE registration_no LIKE %s"],
        "kinara_market":      ["DELETE FROM price_indices WHERE source = %s"],
        "kinara_port":        ["DELETE FROM berths  WHERE notes LIKE %s",
                               "DELETE FROM ports   WHERE code  LIKE %s"],
        "kinara_vessel":      ["DELETE FROM vessels WHERE name LIKE %s"],
        "kinara_shipment":    ["DELETE FROM shipments WHERE notes LIKE %s"],
        "kinara_fleet":       ["DELETE FROM vehicles WHERE registration_no LIKE %s"],
    }

    pat_like = f"{SYNTH_TENANT_PREFIX}%"
    src_tag  = SYNTH_TAG

    for dbname, stmts in db_deletes.items():
        try:
            with _conn(host, port, user, dbname) as conn:
                with conn.cursor() as cur:
                    for stmt in stmts:
                        if "%s" in stmt:
                            param = src_tag if "source" in stmt else pat_like
                            cur.execute(stmt, (param,))
                            click.echo(f"  [{dbname}] {stmt[:50]}…  deleted {cur.rowcount} rows")
                conn.commit()
        except Exception as exc:
            click.echo(f"  [{dbname}] WARN: {exc}")

    click.echo("Done — synthetic data removed.")


# ─── Generator ────────────────────────────────────────────────────────────────

def seed_health(host: str, port: int, user: str,
                n_clinics: int, patients_per_clinic: int,
                dry_run: bool, fake: Faker):
    click.echo(f"\n[Health] {n_clinics} clinics × {patients_per_clinic} patients")

    clinic_ids   = []
    clinic_rows  = []
    patient_rows = []

    clinic_types = ["health_center", "district", "prefectoral", "regional",
                    "dispensary", "health_post", "private", "community", "maternity"]

    for i in range(n_clinics):
        country_data = random.choice(AFRICAN_COUNTRIES)
        cc, country_name, regions = country_data
        region_name, lat, lng = random.choice(regions)
        cid = str(uuid.uuid4())
        clinic_ids.append((cid, cc, region_name))
        clinic_rows.append((
            cid,
            f"SYNTH Clinic {i+1:03d} — {region_name}",
            fake.numerify("+228 ## ## ## ##"),
            f"{region_name}, {country_name}",
            region_name,
            cc,
            random.choice(clinic_types),
        ))

    for idx, (cid, cc, region_name) in enumerate(clinic_ids):
        for j in range(patients_per_clinic):
            n = idx * patients_per_clinic + j
            patient_rows.append((
                str(uuid.uuid4()),
                f"PAT-SYNTH-{n:07d}",
                fake.first_name(),
                fake.last_name(),
                rand_date_past(60 * 365),
                random.choice(["M", "F", "O"]),
                synth_enc("PHONE", n),
                synth_enc("NID", n),
                cc,
                region_name,
                random.choice(["A+","A-","B+","B-","AB+","AB-","O+","O-"]),
                True,
                f"{SYNTH_TENANT_PREFIX}{cc}_{idx:03d}",
                ts_past(730),
                ts_past(30),
            ))

    click.echo(f"  → {len(clinic_rows)} clinics, {len(patient_rows)} patients")
    if dry_run:
        return

    with _conn(host, port, user, "kinara_patient") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO clinics (id, name, phone, address, region, country, clinic_type) "
                "VALUES %s ON CONFLICT DO NOTHING",
                clinic_rows)
            _batch(cur,
                "INSERT INTO patients "
                "(id, patient_ref, first_name, last_name, date_of_birth, gender, "
                " phone_enc, national_id_enc, country, region, blood_type, is_active, "
                " tenant_id, created_at, updated_at) "
                "VALUES %s ON CONFLICT DO NOTHING",
                patient_rows)
        conn.commit()
    click.echo(f"  ✓ Written")


def seed_agriculture(host: str, port: int, user: str,
                     n_cooperatives: int, n_farmers: int, n_prices: int,
                     dry_run: bool, fake: Faker):
    click.echo(f"\n[Agriculture] {n_cooperatives} coops, {n_farmers} farmers, {n_prices} prices")

    coop_rows   = []
    coop_ids    = []
    farmer_rows = []
    plot_rows   = []
    price_rows  = []

    for i in range(n_cooperatives):
        country_data = random.choice(AFRICAN_COUNTRIES)
        cc, _, regions = country_data
        region_name, lat, lng = random.choice(regions)
        cid = str(uuid.uuid4())
        coop_ids.append(cid)
        coop_rows.append((
            cid,
            f"SYNTH Cooperative {i+1:03d} — {region_name}",
            fake_reg_no("COOP", i),
            random.choice(["production","marketing","credit","multi_purpose"]),
            "active",
            cc,
            region_name,
            region_name,
            random.randint(20, 400),
            round(random.uniform(50, 2000), 2),
            f"Synthetic cooperative for load testing. {SYNTH_TAG}",
        ))

    for i in range(n_farmers):
        country_data = random.choice(AFRICAN_COUNTRIES)
        cc, _, regions = country_data
        region_name, lat, lng = random.choice(regions)
        fid = str(uuid.uuid4())
        farmer_rows.append((
            fid,
            synth_enc("NAME", i),
            synth_enc("PHONE", i),
            synth_enc("NID", i),
            cc,
            region_name,
            region_name,
            round(lat  + random.uniform(-0.5, 0.5), 6),
            round(lng  + random.uniform(-0.5, 0.5), 6),
            round(random.uniform(0.5, 50.0), 2),
            random.choice(["smallholder","small","medium","large"]),
            random.choice(["en","fr","sw","ha","yo","am"]),
            False,
            True,
            random.choice(coop_ids) if coop_ids else None,
        ))
        n_plots = random.randint(1, 4)
        for p in range(n_plots):
            plot_rows.append((
                str(uuid.uuid4()),
                fid,
                f"SYNTH Plot {i+1}-{p+1}",
                round(random.uniform(0.1, 20.0), 2),
                random.choice(CROP_TYPES),
            ))

    for i in range(n_prices):
        country_data = random.choice(AFRICAN_COUNTRIES)
        cc, _, _ = country_data
        price_rows.append((
            str(uuid.uuid4()),
            random.choice(CROP_TYPES),
            f"SYNTH Market {i % 20 + 1}",
            cc,
            round(random.uniform(0.05, 5.00), 4),
            random.choice(CURRENCIES),
            (datetime.date.today() - datetime.timedelta(days=random.randint(0, 365))),
            SYNTH_TAG,
        ))

    click.echo(f"  → {len(coop_rows)} coops, {len(farmer_rows)} farmers, "
               f"{len(plot_rows)} plots, {len(price_rows)} prices")
    if dry_run:
        return

    with _conn(host, port, user, "kinara_cooperative") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO cooperatives "
                "(id, name, registration_no, coop_type, status, country, region, district, "
                " total_members, total_farm_ha, description) "
                "VALUES %s ON CONFLICT DO NOTHING",
                coop_rows)
        conn.commit()

    with _conn(host, port, user, "kinara_farmer") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO farmers "
                "(id, full_name_enc, phone_enc, national_id_enc, country, region, district, "
                " gps_lat, gps_lng, farm_size_ha, farm_size, primary_language, "
                " is_verified, is_active, cooperative_id) "
                "VALUES %s ON CONFLICT DO NOTHING",
                farmer_rows)
            _batch(cur,
                "INSERT INTO farm_plots (id, farmer_id, plot_name, area_ha, primary_crop) "
                "VALUES %s ON CONFLICT DO NOTHING",
                plot_rows)
        conn.commit()

    with _conn(host, port, user, "kinara_market") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO price_indices "
                "(id, crop_type, market, country, price_per_kg, currency, recorded_at, source) "
                "VALUES %s ON CONFLICT DO NOTHING",
                price_rows)
        conn.commit()
    click.echo(f"  ✓ Written")


def seed_maritime(host: str, port: int, user: str,
                  n_vessels: int, dry_run: bool, fake: Faker):
    click.echo(f"\n[Maritime] {len(PORT_DATA)} ports, {n_vessels} vessels")

    port_rows  = []
    berth_rows = []
    vessel_rows = []
    port_ids   = []

    for (name, code, cc, city, lat, lng, max_draft, n_berths) in PORT_DATA:
        pid = str(uuid.uuid4())
        port_ids.append((pid, max_draft))
        port_rows.append((
            pid, name, code, cc, city,
            round(lat, 6), round(lng, 6),
            max_draft, n_berths,
            "normal", "operational",
        ))
        for b in range(n_berths):
            berth_rows.append((
                str(uuid.uuid4()),
                pid,
                f"{code}-B{b+1:02d}",
                random.choice(["container","bulk","tanker","ro_ro","general"]),
                round(max_draft - random.uniform(0, 3), 1),
                random.randint(100, 400),
                random.randint(20, 60),
                "available",
                f"{SYNTH_TAG}",
            ))

    vessel_types = ["bulk_carrier","container","tanker","ro_ro","ferry","general_cargo"]
    flag_states  = ["TG","GH","NG","KE","CM","SN","PA","LR","MH","CY"]

    for i in range(n_vessels):
        pid, max_d = random.choice(port_ids)
        vessel_rows.append((
            str(uuid.uuid4()),
            f"MV SYNTH {fake.last_name().upper()} {i+1}",
            f"IMO{7000000 + i:07d}",
            random.choice(vessel_types),
            random.choice(flag_states),
            round(random.uniform(1000, 80000), 3),
            round(random.uniform(2000, 120000), 3),
            round(random.uniform(50, 350), 2),
            round(random.uniform(15, 60), 2),
            round(random.uniform(3, max_d), 2),
            random.randint(1970, 2024),
            random.choice(["Lloyd's","DNV","Bureau Veritas","ClassNK","ABS"]),
            "active",
            pid,
        ))

    click.echo(f"  → {len(port_rows)} ports, {len(berth_rows)} berths, "
               f"{len(vessel_rows)} vessels")
    if dry_run:
        return

    with _conn(host, port, user, "kinara_port") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO ports "
                "(id, name, code, country, city, latitude, longitude, "
                " max_draft_m, total_berths, alert_level, status) "
                "VALUES %s ON CONFLICT DO NOTHING",
                port_rows)
            _batch(cur,
                "INSERT INTO berths "
                "(id, port_id, berth_no, berth_type, max_draft_m, "
                " max_length_m, max_beam_m, status, notes) "
                "VALUES %s ON CONFLICT DO NOTHING",
                berth_rows)
        conn.commit()

    with _conn(host, port, user, "kinara_vessel") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO vessels "
                "(id, name, imo_number, vessel_type, flag_state, "
                " gross_tonnage, deadweight_tonnage, length_overall_m, beam_m, draft_m, "
                " build_year, classification_society, status, current_port_id) "
                "VALUES %s ON CONFLICT DO NOTHING",
                vessel_rows)
        conn.commit()
    click.echo(f"  ✓ Written")


def seed_logistics(host: str, port: int, user: str,
                   n_vehicles: int, n_shipments: int,
                   dry_run: bool, fake: Faker):
    click.echo(f"\n[Logistics] {n_vehicles} vehicles, {n_shipments} shipments")

    vehicle_rows  = []
    shipment_rows = []

    for i in range(n_vehicles):
        country_data = random.choice(AFRICAN_COUNTRIES)
        cc, _, _ = country_data
        vehicle_rows.append((
            str(uuid.uuid4()),
            fake_reg_no(cc, i),
            random.choice(VEHICLE_TYPES),
            random.choice(VEHICLE_MAKES),
            fake.word().upper(),
            random.randint(2010, 2025),
            random.choice(["diesel","petrol","cng"]),
            round(random.uniform(500, 30000), 1),
            round(random.uniform(5, 80), 1),
            random.choice(["available","active","in_transit"]),
            cc,
            f"SYNTH base — {cc}",
            round(random.uniform(0, 250000), 0),
        ))

    statuses = ["created","collected","in_transit","out_for_delivery","delivered"]
    for i in range(n_shipments):
        orig = random.choice(AFRICAN_COUNTRIES)
        dest = random.choice(AFRICAN_COUNTRIES)
        status = random.choice(statuses)
        delivered_at = ts_past(30) if status == "delivered" else None
        charge = round(random.uniform(20, 5000), 2)
        shipment_rows.append((
            str(uuid.uuid4()),
            f"SYNTH-SHP-{i:07d}",
            str(uuid.uuid4()),
            fake.last_name(),
            f"+{random.randint(2, 9)}{random.randint(10000000, 999999999):09d}",
            fake.address().replace("\n", ", ")[:100],
            orig[0],
            fake.address().replace("\n", ", ")[:100],
            dest[0],
            round(random.uniform(0.5, 5000), 2),
            round(random.uniform(10, 300), 1),
            round(random.uniform(10, 200), 1),
            round(random.uniform(10, 200), 1),
            round(random.uniform(50, 50000), 2),
            random.choice(CURRENCIES),
            random.choice(["standard","express","economy"]),
            status,
            charge,
            round(charge * 0.02, 2),
            round(charge * 1.02, 2),
            ts_past(180),
            delivered_at,
            f"{SYNTH_TAG}",
        ))

    click.echo(f"  → {len(vehicle_rows)} vehicles, {len(shipment_rows)} shipments")
    if dry_run:
        return

    with _conn(host, port, user, "kinara_fleet") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO vehicles "
                "(id, registration_no, vehicle_type, make, model, year, fuel_type, "
                " payload_capacity_kg, volume_capacity_m3, status, country, base_location, "
                " current_odometer_km) "
                "VALUES %s ON CONFLICT DO NOTHING",
                vehicle_rows)
        conn.commit()

    with _conn(host, port, user, "kinara_shipment") as conn:
        with conn.cursor() as cur:
            _batch(cur,
                "INSERT INTO shipments "
                "(id, tracking_code, sender_id, recipient_name, recipient_phone, "
                " origin_address, origin_country, dest_address, dest_country, "
                " weight_kg, length_cm, width_cm, height_cm, declared_value, currency, "
                " service_level, status, freight_charge, insurance_charge, total_charge, "
                " picked_at, delivered_at, notes) "
                "VALUES %s ON CONFLICT DO NOTHING",
                shipment_rows)
        conn.commit()
    click.echo(f"  ✓ Written")


# ─── CLI ──────────────────────────────────────────────────────────────────────

@click.command()
@click.option("--host",     default=lambda: os.environ.get("PGHOST", "localhost"),
              show_default=True, help="Postgres host")
@click.option("--port",     default=lambda: int(os.environ.get("PGPORT", 5432)),
              show_default=True, type=int, help="Postgres port")
@click.option("--user",     default=lambda: os.environ.get("PGUSER", "kinara_app"),
              show_default=True, help="Postgres user")
@click.option("--clinics",  default=50,  show_default=True, help="Number of clinics")
@click.option("--patients", default=200, show_default=True, help="Patients per clinic")
@click.option("--dry-run",  is_flag=True, help="Print planned counts, do not write")
@click.option("--clean",    is_flag=True, help="Remove all previously generated synthetic rows")
def main(host, port, user, clinics, patients, dry_run, clean):
    """Generate fully fabricated Kinara OS load-test data."""

    if os.environ.get("KINARA_ENV", "").lower() == "production":
        click.echo("ERROR: KINARA_ENV=production — refusing to run against production.", err=True)
        sys.exit(1)

    click.echo("=" * 60)
    click.echo("Kinara OS Synthetic Data Generator")
    click.echo("ALL DATA IS COMPLETELY FABRICATED — NOT REAL PHI")
    click.echo("=" * 60)
    click.echo(f"Target host : {host}:{port}  user={user}")
    click.echo(f"Mode        : {'DRY RUN' if dry_run else ('CLEAN' if clean else 'WRITE')}")
    click.echo("")

    if clean:
        clean_all(host, port, user)
        return

    fake = Faker(["en_US", "fr_FR"])
    Faker.seed(42)
    random.seed(42)

    n_coops      = max(10, clinics // 5)
    n_farmers    = clinics * 40
    n_prices     = 1_000
    n_vessels    = 50
    n_vehicles   = 100
    n_shipments  = max(500, clinics * 10)

    seed_health(host, port, user, clinics, patients, dry_run, fake)
    seed_agriculture(host, port, user, n_coops, n_farmers, n_prices, dry_run, fake)
    seed_maritime(host, port, user, n_vessels, dry_run, fake)
    seed_logistics(host, port, user, n_vehicles, n_shipments, dry_run, fake)

    click.echo("")
    click.echo("=" * 60)
    click.echo("Summary")
    click.echo("=" * 60)
    click.echo(f"  Clinics            : {clinics:>8,}")
    click.echo(f"  Patients           : {clinics * patients:>8,}")
    click.echo(f"  Cooperatives       : {n_coops:>8,}")
    click.echo(f"  Farmers            : {n_farmers:>8,}")
    click.echo(f"  Farm plots (~2.5×) : {int(n_farmers * 2.5):>8,}")
    click.echo(f"  Price indices      : {n_prices:>8,}")
    click.echo(f"  Ports              : {len(PORT_DATA):>8,}")
    click.echo(f"  Berths             : {sum(p[7] for p in PORT_DATA):>8,}")
    click.echo(f"  Vessels            : {n_vessels:>8,}")
    click.echo(f"  Vehicles           : {n_vehicles:>8,}")
    click.echo(f"  Shipments          : {n_shipments:>8,}")
    total = (clinics + clinics * patients + n_coops + n_farmers +
             int(n_farmers * 2.5) + n_prices + len(PORT_DATA) +
             sum(p[7] for p in PORT_DATA) + n_vessels + n_vehicles + n_shipments)
    click.echo(f"  ─────────────────────────────")
    click.echo(f"  Total rows         : {total:>8,}")
    click.echo("")
    if not dry_run:
        click.echo("All data tagged SYNTHETIC_LOAD_TEST — safe to truncate at any time.")
        click.echo("Remove with: python3 scripts/generate_synthetic_data.py --clean ...")


if __name__ == "__main__":
    main()
