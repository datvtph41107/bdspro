BEGIN;
CREATE TABLE permissions (code TEXT PRIMARY KEY CHECK(btrim(code)<>''));
INSERT INTO permissions(code) VALUES
 ('members.read'),('members.invite'),('sources.create'),('sources.read'),('sources.price_report');
CREATE TABLE organization_roles (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
 name TEXT NOT NULL CHECK (btrim(name)<>''),
 UNIQUE(organization_id,id),UNIQUE(organization_id,name)
);
CREATE TABLE organization_role_permissions (
 role_id BIGINT NOT NULL REFERENCES organization_roles(id),
 permission_code TEXT NOT NULL REFERENCES permissions(code),
 PRIMARY KEY(role_id,permission_code)
);
CREATE TABLE organization_memberships (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
 person_id BIGINT NOT NULL REFERENCES persons(party_id),
 joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 ended_at TIMESTAMPTZ,
 UNIQUE(organization_id,id),
 CHECK(ended_at IS NULL OR ended_at>=joined_at)
);
CREATE UNIQUE INDEX organization_memberships_one_open_uq
 ON organization_memberships(organization_id,person_id) WHERE ended_at IS NULL;
CREATE TABLE organization_membership_blocks (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 membership_id BIGINT NOT NULL REFERENCES organization_memberships(id),
 blocked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 ended_at TIMESTAMPTZ,
 CHECK(ended_at IS NULL OR ended_at>=blocked_at)
);
CREATE UNIQUE INDEX organization_membership_blocks_one_open_uq
 ON organization_membership_blocks(membership_id) WHERE ended_at IS NULL;
CREATE TABLE organization_role_assignments (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT NOT NULL,
 membership_id BIGINT NOT NULL,
 role_id BIGINT NOT NULL,
 assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 ended_at TIMESTAMPTZ,
 CHECK(ended_at IS NULL OR ended_at>=assigned_at),
 FOREIGN KEY(organization_id,membership_id)
   REFERENCES organization_memberships(organization_id,id),
 FOREIGN KEY(organization_id,role_id)
   REFERENCES organization_roles(organization_id,id)
);
CREATE UNIQUE INDEX organization_role_assignments_one_open_uq
 ON organization_role_assignments(membership_id,role_id) WHERE ended_at IS NULL;
CREATE TABLE organization_invitations (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id BIGINT NOT NULL REFERENCES organizations(party_id),
 invitee_email TEXT NOT NULL CHECK(btrim(invitee_email)<>''),
 intended_role_id BIGINT NOT NULL,
 invited_by_account_id BIGINT NOT NULL REFERENCES accounts(id),
 invited_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ NOT NULL,
 outcome TEXT,
 resolved_at TIMESTAMPTZ,
 CHECK(expires_at>invited_at),
 CHECK ((outcome IS NULL AND resolved_at IS NULL) OR
        (outcome IN ('ACCEPTED','DECLINED','CANCELLED') AND resolved_at IS NOT NULL)),
 UNIQUE(organization_id,id),
 FOREIGN KEY(organization_id,intended_role_id)
   REFERENCES organization_roles(organization_id,id)
);
ALTER TABLE organization_memberships ADD COLUMN originating_invitation_id BIGINT UNIQUE;
ALTER TABLE organization_memberships ADD CONSTRAINT memberships_invitation_scope_fk
 FOREIGN KEY(organization_id,originating_invitation_id)
 REFERENCES organization_invitations(organization_id,id);
CREATE TABLE organization_primary_administrators (
 organization_id BIGINT PRIMARY KEY REFERENCES organizations(party_id),
 membership_id BIGINT NOT NULL UNIQUE,
 FOREIGN KEY(organization_id,membership_id)
   REFERENCES organization_memberships(organization_id,id)
);
COMMIT;
