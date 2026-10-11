-- CORE-05 module-relative migration; requires canonical Auth/Organization tables.
-- NOT the next application migration number until local prerequisite history is reconciled.
BEGIN;

INSERT INTO permissions (code)
VALUES ('sources.create'), ('sources.read'), ('sources.price_report')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE source_intakes (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
  recorded_by_account_id BIGINT NOT NULL REFERENCES accounts(id),
  source_channel_code TEXT NOT NULL CHECK (btrim(source_channel_code) <> ''),
  source_label TEXT,
  reported_area_m2 NUMERIC(18,4) CHECK (reported_area_m2 > 0),
  area_basis_code TEXT,
  received_at TIMESTAMPTZ,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  summary TEXT,
  version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
  archived_at TIMESTAMPTZ,
  capture_request_key TEXT NOT NULL CHECK (
    length(capture_request_key) BETWEEN 1 AND 128
  ),
  capture_request_hash TEXT NOT NULL CHECK (
    capture_request_hash ~ '^[0-9a-f]{64}$'
  ),
  CONSTRAINT source_intakes_org_id_uq UNIQUE (organization_id, id),
  CONSTRAINT source_intakes_request_uq UNIQUE (
    organization_id, recorded_by_account_id, capture_request_key
  )
);

CREATE TABLE source_price_reports (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
  intake_id BIGINT NOT NULL,
  recorded_by_account_id BIGINT NOT NULL REFERENCES accounts(id),
  amount NUMERIC(19,4) NOT NULL CHECK (amount > 0),
  currency_code VARCHAR(3) NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
  price_basis_code TEXT,
  source_basis_code TEXT,
  statement_at TIMESTAMPTZ,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  note TEXT,
  request_key TEXT,
  request_hash TEXT,
  CONSTRAINT source_price_reports_key_pair_ck CHECK (
    (request_key IS NULL AND request_hash IS NULL)
    OR
    (
      request_key IS NOT NULL
      AND length(request_key) BETWEEN 1 AND 128
      AND request_hash ~ '^[0-9a-f]{64}$'
    )
  ),
  CONSTRAINT source_price_reports_request_uq UNIQUE (
    organization_id, intake_id, recorded_by_account_id, request_key
  ),
  CONSTRAINT source_price_reports_intake_scope_fk FOREIGN KEY (
    organization_id, intake_id
  ) REFERENCES source_intakes (organization_id, id)
);

CREATE INDEX source_price_reports_history_idx
  ON source_price_reports (organization_id, intake_id, recorded_at, id);

COMMIT;
