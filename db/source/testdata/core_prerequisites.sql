-- TEST/SQLC FIXTURE ONLY. NOT a production Identity/Auth migration.
-- Recreates the small shape of the canonical tables CORE-05 needs.
CREATE TABLE parties (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);
CREATE TABLE persons (
  party_id BIGINT PRIMARY KEY REFERENCES parties(id)
);
CREATE TABLE organizations (
  party_id BIGINT PRIMARY KEY REFERENCES parties(id),
  name TEXT NOT NULL,
  archived_at TIMESTAMPTZ
);
CREATE TABLE accounts (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  person_id BIGINT NOT NULL REFERENCES persons(party_id),
  closed_at TIMESTAMPTZ
);
CREATE TABLE permissions (code TEXT PRIMARY KEY);
CREATE TABLE organization_roles (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
  name TEXT NOT NULL,
  UNIQUE (organization_id, id)
);
CREATE TABLE organization_role_permissions (
  role_id BIGINT NOT NULL REFERENCES organization_roles(id),
  permission_code TEXT NOT NULL REFERENCES permissions(code),
  PRIMARY KEY (role_id, permission_code)
);
CREATE TABLE organization_memberships (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
  person_id BIGINT NOT NULL REFERENCES persons(party_id),
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ,
  UNIQUE (organization_id, id)
);
CREATE TABLE organization_role_assignments (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id BIGINT NOT NULL,
  membership_id BIGINT NOT NULL,
  role_id BIGINT NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ,
  FOREIGN KEY (organization_id,membership_id)
    REFERENCES organization_memberships(organization_id,id),
  FOREIGN KEY (organization_id,role_id)
    REFERENCES organization_roles(organization_id,id)
);
CREATE TABLE organization_membership_blocks (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  membership_id BIGINT NOT NULL REFERENCES organization_memberships(id),
  blocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ
);
