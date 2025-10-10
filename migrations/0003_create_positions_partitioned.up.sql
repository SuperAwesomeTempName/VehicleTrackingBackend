-- Parent table for time partitioning
CREATE TABLE IF NOT EXISTS positions (
  id bigserial,
  bus_id uuid NOT NULL REFERENCES buses(id),
  route_id uuid,
  ts timestamptz NOT NULL,
  speed_kph double precision,
  heading double precision,
  geom geometry(Point,4326) NOT NULL,
  raw jsonb,
  created_at timestamptz DEFAULT now(),
  PRIMARY KEY (id, ts)
) PARTITION BY RANGE (ts);

-- Function to create monthly partitions 
CREATE OR REPLACE FUNCTION create_positions_partition(year int, month int) RETURNS void AS $$
DECLARE
  start_date date := make_date(year, month, 1);
  end_date date := (start_date + interval '1 month')::date;
  partition_name text := format('positions_%s_%s', year, lpad(month::text,2,'0'));
BEGIN
  EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF positions FOR VALUES FROM (%L) TO (%L);', partition_name, start_date, end_date);
  EXECUTE format('CREATE INDEX IF NOT EXISTS %I_geom_idx ON %I USING GIST (geom);', partition_name || '_geom_idx', partition_name);
END;
$$ LANGUAGE plpgsql;

-- Initial partition for current month
SELECT create_positions_partition(EXTRACT(YEAR FROM now())::int, EXTRACT(MONTH FROM now())::int);

CREATE INDEX IF NOT EXISTS idx_positions_bus_ts ON positions (bus_id, ts DESC);






