#!/usr/bin/env python3
"""
Apply the scaffold tenant-isolation fix and generate handler_test.go for all
scaffold services.  Run from the repo root:

    python3 scripts/gen_scaffold_tests.py

What it does for each scaffold service (those with DataEnc in db/db.go):
  1. Rewrites db/db.go — adds TenantID, updates Querier interface + SQL.
  2. Rewrites handlers/handler.go — plumbs tenantID through list/create/get/delete.
  3. Writes handlers/handler_test.go from the mental-health-service template.

Skips:
  - mental-health-service (already done manually, is the source template)
  - health-worker-service (diverged New() signature — has extra Publisher arg)
  - Any service whose handlers/handler.go has already been updated (idempotent).
"""

import os
import re
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SERVICES = os.path.join(REPO, "services")
TEMPLATE_SVC = "mental-health-service"
SKIP = {TEMPLATE_SVC, "health-worker-service"}

# ── db.go template ────────────────────────────────────────────────────────────

DB_QUERIER_OLD = '''\
type Queries struct {
\tpool *pgxpool.Pool
}'''

DB_QUERIER_NEW_TMPL = '''\
type Querier interface {
\tCreate(ctx context.Context, r Record) error
\tGet(ctx context.Context, id uuid.UUID) (*Record, error)
\tList(ctx context.Context, limit, offset int, tenantID uuid.UUID) ([]Record, error)
\tDelete(ctx context.Context, id, tenantID uuid.UUID) error
}

type Queries struct {
\tpool *pgxpool.Pool
}'''

DB_RECORD_OLD = '''\
type Record struct {
\tID        uuid.UUID  `db:"id"`
\tDataEnc   string     `db:"data_enc"`
\tCreatedBy uuid.UUID  `db:"created_by"`
\tCreatedAt time.Time  `db:"created_at"`
\tUpdatedAt time.Time  `db:"updated_at"`
}'''

DB_RECORD_NEW = '''\
type Record struct {
\tID        uuid.UUID `db:"id"         json:"id"`
\tDataEnc   string    `db:"data_enc"   json:"data_enc"`
\tCreatedBy uuid.UUID `db:"created_by" json:"created_by"`
\tTenantID  uuid.UUID `db:"tenant_id"  json:"tenant_id"`
\tCreatedAt time.Time `db:"created_at" json:"created_at"`
\tUpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}'''

DB_CREATE_OLD = '''\
func (q *Queries) Create(ctx context.Context, r Record) error {
\t_, err := q.pool.Exec(ctx,
\t\t`INSERT INTO records (id, data_enc, created_by, created_at, updated_at)
\t\t VALUES ($1, $2, $3, $4, $5)`,
\t\tr.ID, r.DataEnc, r.CreatedBy, r.CreatedAt, r.UpdatedAt)
\treturn err
}'''

DB_CREATE_NEW = '''\
func (q *Queries) Create(ctx context.Context, r Record) error {
\t_, err := q.pool.Exec(ctx,
\t\t`INSERT INTO records (id, data_enc, created_by, tenant_id, created_at, updated_at)
\t\t VALUES ($1, $2, $3, $4, $5, $6)`,
\t\tr.ID, r.DataEnc, r.CreatedBy, r.TenantID, r.CreatedAt, r.UpdatedAt)
\treturn err
}'''

DB_GET_OLD = '''\
func (q *Queries) Get(ctx context.Context, id uuid.UUID) (*Record, error) {
\tvar r Record
\terr := q.pool.QueryRow(ctx,
\t\t`SELECT id, data_enc, created_by, created_at, updated_at FROM records WHERE id = $1`, id).
\t\tScan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
\tif err != nil {
\t\treturn nil, err
\t}
\treturn &r, nil
}'''

DB_GET_NEW = '''\
func (q *Queries) Get(ctx context.Context, id uuid.UUID) (*Record, error) {
\tvar r Record
\terr := q.pool.QueryRow(ctx,
\t\t`SELECT id, data_enc, created_by, tenant_id, created_at, updated_at FROM records WHERE id = $1`, id).
\t\tScan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.TenantID, &r.CreatedAt, &r.UpdatedAt)
\tif err != nil {
\t\treturn nil, err
\t}
\treturn &r, nil
}'''

DB_LIST_OLD = '''\
func (q *Queries) List(ctx context.Context, limit, offset int) ([]Record, error) {
\trows, err := q.pool.Query(ctx,
\t\t`SELECT id, data_enc, created_by, created_at, updated_at FROM records
\t\t ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar result []Record
\tfor rows.Next() {
\t\tvar r Record
\t\tif err := rows.Scan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tresult = append(result, r)
\t}
\treturn result, rows.Err()
}'''

DB_LIST_NEW = '''\
func (q *Queries) List(ctx context.Context, limit, offset int, tenantID uuid.UUID) ([]Record, error) {
\trows, err := q.pool.Query(ctx,
\t\t`SELECT id, data_enc, created_by, tenant_id, created_at, updated_at FROM records
\t\t WHERE tenant_id = $3 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset, tenantID)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar result []Record
\tfor rows.Next() {
\t\tvar r Record
\t\tif err := rows.Scan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.TenantID, &r.CreatedAt, &r.UpdatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tresult = append(result, r)
\t}
\treturn result, rows.Err()
}'''

DB_DELETE_OLD = '''\
func (q *Queries) Delete(ctx context.Context, id uuid.UUID) error {
\t_, err := q.pool.Exec(ctx, `DELETE FROM records WHERE id = $1`, id)
\treturn err
}'''

DB_DELETE_NEW = '''\
func (q *Queries) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
\t_, err := q.pool.Exec(ctx, `DELETE FROM records WHERE id = $1 AND tenant_id = $2`, id, tenantID)
\treturn err
}'''

# ── helpers ───────────────────────────────────────────────────────────────────

_DB_TEMPLATE = None

def get_db_template():
    global _DB_TEMPLATE
    if _DB_TEMPLATE is None:
        with open(os.path.join(SERVICES, TEMPLATE_SVC, "db", "db.go")) as f:
            _DB_TEMPLATE = f.read()
    return _DB_TEMPLATE

def apply_db_fix(path):
    # All scaffold db.go files are structurally identical (same package/imports/types).
    # Copy the corrected template directly rather than applying text patches.
    template = get_db_template()
    with open(path) as f:
        current = f.read()
    if current == template:
        return False  # already correct
    with open(path, "w") as f:
        f.write(template)
    return True


# handler.go patches are done via regex on the three methods that need changing.

def apply_handler_fix(path):
    with open(path) as f:
        src = f.read()
    if "claims.TenantID" in src and "db.Querier" in src:
        return False  # already updated

    # 0. Change Handler.queries type and New() signature to use db.Querier
    src = re.sub(r'queries \*db\.Queries', 'queries db.Querier', src)
    src = re.sub(r'func New\(q \*db\.Queries,', 'func New(q db.Querier,', src)

    # 1. list — add claims line and pass tenantID
    src = src.replace(
        "func (h *Handler) list(w http.ResponseWriter, r *http.Request) {\n"
        "\tpage, _ := strconv.Atoi(r.URL.Query().Get(\"page\"))\n",
        "func (h *Handler) list(w http.ResponseWriter, r *http.Request) {\n"
        "\tclaims := middleware.ClaimsFromContext(r.Context())\n"
        "\tpage, _ := strconv.Atoi(r.URL.Query().Get(\"page\"))\n",
    )
    # list query — old vs new call
    src = re.sub(
        r'h\.queries\.List\(r\.Context\(\), 20, \(page-1\)\*20\)',
        'h.queries.List(r.Context(), 20, (page-1)*20, claims.TenantID)',
        src,
    )

    # 2. create — add TenantID to rec literal
    src = src.replace(
        "\t\tCreatedBy: claims.UserID,\n"
        "\t\tCreatedAt: now,\n",
        "\t\tCreatedBy: claims.UserID,\n"
        "\t\tTenantID:  claims.TenantID,\n"
        "\t\tCreatedAt: now,\n",
    )

    # 3. get — add claims + tenant check after rec fetch
    src = src.replace(
        "func (h *Handler) get(w http.ResponseWriter, r *http.Request) {\n"
        "\tid, err := uuid.Parse(mux.Vars(r)[\"id\"])\n",
        "func (h *Handler) get(w http.ResponseWriter, r *http.Request) {\n"
        "\tclaims := middleware.ClaimsFromContext(r.Context())\n"
        "\tid, err := uuid.Parse(mux.Vars(r)[\"id\"])\n",
    )
    src = src.replace(
        "\trec, err := h.queries.Get(r.Context(), id)\n"
        "\tif err != nil {\n"
        "\t\th.notFound(w)\n"
        "\t\treturn\n"
        "\t}\n"
        "\tplain, err",
        "\trec, err := h.queries.Get(r.Context(), id)\n"
        "\tif err != nil {\n"
        "\t\th.notFound(w)\n"
        "\t\treturn\n"
        "\t}\n"
        "\t// Return 404 (not 403) — avoids leaking that the record exists to another tenant.\n"
        "\tif rec.TenantID != claims.TenantID {\n"
        "\t\th.notFound(w)\n"
        "\t\treturn\n"
        "\t}\n"
        "\tplain, err",
    )

    # 4. delete — add claims + tenantID to Delete call
    src = src.replace(
        "\tif err := h.queries.Delete(r.Context(), id); err != nil {\n",
        "\tclaims := middleware.ClaimsFromContext(r.Context())\n"
        "\tif err := h.queries.Delete(r.Context(), id, claims.TenantID); err != nil {\n",
    )

    with open(path, "w") as f:
        f.write(src)
    return True


UPDATE_NOTE = (
    "\t// NOTE: unimplemented stub — returns 200 without persisting anything or checking\n"
    "\t// ownership. When implementing, fetch the record and apply the same tenant guard\n"
    "\t// the get handler uses (if rec.TenantID != claims.TenantID -> h.notFound(w)),\n"
    "\t// otherwise this endpoint becomes a cross-tenant write.\n"
)


def apply_update_note(path):
    """Insert the stub note as the first line of the update handler body. Idempotent.

    Runs independently of apply_handler_fix, which early-returns on already-fixed
    services and would otherwise never reach the note insertion.
    """
    with open(path) as f:
        src = f.read()
    if "unimplemented stub" in src:
        return False
    anchor = "func (h *Handler) update(w http.ResponseWriter, r *http.Request) {\n"
    if anchor not in src:
        return False
    src = src.replace(anchor, anchor + UPDATE_NOTE, 1)
    with open(path, "w") as f:
        f.write(src)
    return True


def get_resource_path(handler_path):
    with open(handler_path) as f:
        content = f.read()
    m = re.search(r'api\.HandleFunc\("(/[^"]*)"', content)
    return "/api/v1" + m.group(1) if m else None


def read_template():
    tpl_path = os.path.join(SERVICES, TEMPLATE_SVC, "handlers", "handler_test.go")
    with open(tpl_path) as f:
        return f.read()


def read_isolation_template():
    tpl_path = os.path.join(SERVICES, TEMPLATE_SVC, "handlers", "isolation_test.go")
    with open(tpl_path) as f:
        return f.read()


def generate_test(svc, resource_path, template):
    content = template.replace(TEMPLATE_SVC, svc)
    content = content.replace("/api/v1/mental-healths", resource_path)
    return content


# ── main ──────────────────────────────────────────────────────────────────────

def main():
    template = read_template()
    isolation_template = read_isolation_template()
    updated_db = 0
    updated_handler = 0
    updated_note = 0
    generated_tests = 0
    generated_isolation = 0
    skipped = []

    for svc in sorted(os.listdir(SERVICES)):
        if svc in SKIP:
            continue
        svc_dir = os.path.join(SERVICES, svc)
        db_path = os.path.join(svc_dir, "db", "db.go")
        handler_path = os.path.join(svc_dir, "handlers", "handler.go")
        test_path = os.path.join(svc_dir, "handlers", "handler_test.go")

        if not os.path.exists(db_path) or not os.path.exists(handler_path):
            continue

        with open(db_path) as f:
            db_content = f.read()
        if "DataEnc" not in db_content:
            continue  # not a scaffold service

        resource_path = get_resource_path(handler_path)
        if not resource_path:
            skipped.append(f"{svc} (no resource path detected)")
            continue

        if apply_db_fix(db_path):
            updated_db += 1

        if apply_handler_fix(handler_path):
            updated_handler += 1

        if apply_update_note(handler_path):
            updated_note += 1

        test_content = generate_test(svc, resource_path, template)
        with open(test_path, "w") as f:
            f.write(test_content)
        generated_tests += 1

        isolation_path = os.path.join(svc_dir, "handlers", "isolation_test.go")
        isolation_content = generate_test(svc, resource_path, isolation_template)
        with open(isolation_path, "w") as f:
            f.write(isolation_content)
        generated_isolation += 1

    print(f"db.go updated:            {updated_db}")
    print(f"handler.go updated:       {updated_handler}")
    print(f"update-note inserted:     {updated_note}")
    print(f"handler_test.go written:  {generated_tests}")
    print(f"isolation_test.go written:{generated_isolation}")
    if skipped:
        print(f"skipped: {skipped}")
    total = generated_tests + 1  # +1 for mental-health-service
    print(f"\nTotal scaffold services now covered: {total}")


if __name__ == "__main__":
    main()
