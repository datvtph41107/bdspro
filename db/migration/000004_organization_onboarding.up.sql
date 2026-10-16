BEGIN;
-- Provision reviewers ONLY via a privileged operator path, never public API.
CREATE TABLE platform_organization_reviewers (
 account_id BIGINT PRIMARY KEY REFERENCES accounts(id),
 authorized_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 revoked_at TIMESTAMPTZ,
 authorization_reference TEXT NOT NULL CHECK(btrim(authorization_reference)<>'')
);
CREATE TABLE organization_registration_requests (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 requested_by_account_id BIGINT NOT NULL REFERENCES accounts(id),
 declared_name TEXT NOT NULL CHECK(btrim(declared_name)<>''),
 declared_jurisdiction_code TEXT NOT NULL CHECK(btrim(declared_jurisdiction_code)<>''),
 declared_registration_code TEXT,
 submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 matched_organization_id BIGINT REFERENCES organizations(party_id),
 resolved_at TIMESTAMPTZ,
 decision_code TEXT,
 reviewed_by_account_id BIGINT REFERENCES platform_organization_reviewers(account_id),
 review_reference TEXT,
 CHECK (
   (decision_code IS NULL AND resolved_at IS NULL AND reviewed_by_account_id IS NULL AND matched_organization_id IS NULL AND review_reference IS NULL)
   OR
   (decision_code IN ('APPROVED','REJECTED','MATCHED_EXISTING') AND resolved_at IS NOT NULL
    AND reviewed_by_account_id IS NOT NULL AND NULLIF(btrim(review_reference),'') IS NOT NULL
    AND (decision_code='REJECTED' OR matched_organization_id IS NOT NULL))
 )
);
CREATE TABLE organization_claim_requests (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
 requested_by_account_id BIGINT NOT NULL REFERENCES accounts(id),
 evidence_reference TEXT NOT NULL CHECK(btrim(evidence_reference)<>''),
 submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 resolved_at TIMESTAMPTZ,
 decision_code TEXT,
 reviewed_by_account_id BIGINT REFERENCES platform_organization_reviewers(account_id),
 review_reference TEXT,
 CHECK (
   (decision_code IS NULL AND resolved_at IS NULL AND reviewed_by_account_id IS NULL AND review_reference IS NULL)
   OR
   (decision_code IN ('APPROVED','REJECTED') AND resolved_at IS NOT NULL
    AND reviewed_by_account_id IS NOT NULL AND NULLIF(btrim(review_reference),'') IS NOT NULL)
 )
);
-- Claimed registry ID never proves legal existence; insertion requires human verification.
CREATE TABLE organization_registry_identifiers (
 organization_id BIGINT NOT NULL UNIQUE REFERENCES organizations(party_id),
 jurisdiction_code TEXT NOT NULL CHECK(btrim(jurisdiction_code)<>''),
 registration_code TEXT NOT NULL CHECK(btrim(registration_code)<>''),
 verified_by_account_id BIGINT NOT NULL REFERENCES platform_organization_reviewers(account_id),
 verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(jurisdiction_code,registration_code)
);
COMMIT;
