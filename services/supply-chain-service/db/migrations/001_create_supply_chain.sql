-- Supply chain service schema
-- Bridges agriculture pillar to logistics pillar.

CREATE TABLE IF NOT EXISTS shipments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_ref       TEXT NOT NULL UNIQUE,
    farmer_id          UUID NOT NULL,
    cooperative_id     UUID,
    commodity_name     TEXT NOT NULL,
    quantity_kg        NUMERIC(12,3) NOT NULL,
    origin_location    TEXT NOT NULL,
    destination_location TEXT NOT NULL,
    buyer_id           UUID,
    status             TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','picked_up','in_transit','at_warehouse','delivered','cancelled')),
    pillar_handoff     TEXT NOT NULL DEFAULT 'agri_to_logistics'
                          CHECK (pillar_handoff IN ('agri_to_logistics','logistics_to_port','port_to_market')),
    estimated_cost_usd NUMERIC(10,2) NOT NULL DEFAULT 0,
    actual_cost_usd    NUMERIC(10,2),
    picked_up_at       TIMESTAMPTZ,
    delivered_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tracking_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    status      TEXT NOT NULL,
    location    TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_tracking AS ON UPDATE TO tracking_events DO INSTEAD NOTHING;
CREATE RULE no_delete_tracking AS ON DELETE TO tracking_events DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS supply_audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id UUID,
    actor_id    UUID NOT NULL,
    action      TEXT NOT NULL,
    detail      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_supply_audit AS ON UPDATE TO supply_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_supply_audit AS ON DELETE TO supply_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_shipments_farmer  ON shipments(farmer_id);
CREATE INDEX IF NOT EXISTS idx_shipments_status  ON shipments(status);
CREATE INDEX IF NOT EXISTS idx_tracking_shipment ON tracking_events(shipment_id);
