-- Rename the stored relationship; keep its rows, role values, history and FKs.
-- Earlier migrations are immutable: this applies to both upgraded and fresh databases.
ALTER TABLE organization_memberships RENAME TO organization_members;

ALTER TABLE organization_members RENAME CONSTRAINT organization_memberships_pkey
  TO organization_members_pkey;
ALTER TABLE organization_members RENAME CONSTRAINT organization_memberships_organization_id_fkey
  TO organization_members_organization_id_fkey;
ALTER TABLE organization_members RENAME CONSTRAINT organization_memberships_person_id_fkey
  TO organization_members_person_id_fkey;
ALTER TABLE organization_members RENAME CONSTRAINT organization_memberships_role_check
  TO organization_members_role_check;

ALTER INDEX organization_memberships_one_open_uq
  RENAME TO organization_members_one_open_uq;
ALTER INDEX organization_memberships_person_idx
  RENAME TO organization_members_person_idx;
