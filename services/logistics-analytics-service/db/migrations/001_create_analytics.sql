CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE logistics_metrics (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  period               TEXT NOT NULL,
  period_start         TIMESTAMPTZ NOT NULL,
  period_end           TIMESTAMPTZ NOT NULL,
  country              TEXT NOT NULL,
  total_trips          INT NOT NULL DEFAULT 0,
  total_distance_km    DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_deliveries     INT NOT NULL DEFAULT 0,
  successful_deliveries INT NOT NULL DEFAULT 0,
  on_time_deliveries   INT NOT NULL DEFAULT 0,
  on_time_rate         DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_cost_per_km      DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_cost_per_delivery DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_revenue        DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency             TEXT NOT NULL DEFAULT 'USD',
  bottleneck_route     TEXT,
  bottleneck_warehouse TEXT,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE demand_forecasts (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  country          TEXT NOT NULL,
  route            TEXT NOT NULL,
  forecast_date    TIMESTAMPTZ NOT NULL,
  predicted_volume DOUBLE PRECISION NOT NULL DEFAULT 0,
  predicted_trips  INT NOT NULL DEFAULT 0,
  confidence_pct   DOUBLE PRECISION NOT NULL DEFAULT 0,
  notes            TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE analytics_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_logistics_metrics AS ON UPDATE TO logistics_metrics DO INSTEAD NOTHING;
CREATE RULE no_delete_logistics_metrics AS ON DELETE TO logistics_metrics DO INSTEAD NOTHING;
CREATE RULE no_update_demand_forecasts AS ON UPDATE TO demand_forecasts DO INSTEAD NOTHING;
CREATE RULE no_delete_demand_forecasts AS ON DELETE TO demand_forecasts DO INSTEAD NOTHING;
CREATE RULE no_update_analytics_audit AS ON UPDATE TO analytics_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_analytics_audit AS ON DELETE TO analytics_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_metrics_period ON logistics_metrics(period, country, period_start DESC);
CREATE INDEX idx_forecasts_date ON demand_forecasts(forecast_date, country);
