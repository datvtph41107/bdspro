-- A live organization owner must retain a real Account for that Person.
-- No triggers and no owner-team shadow model. The redundant owner_person_id
-- is a nullable FK to the already unique accounts.person_id, checked to
-- equal the membership person_id for active owner rows only.
ALTER TABLE organization_members
  ADD COLUMN owner_person_id bigint;

-- Fail closed if legacy owner rows have no matching account. Do not silently
-- demote or delete them to make this migration pass.
UPDATE organization_members
SET owner_person_id = person_id
WHERE role = 'owner' AND ended_at IS NULL;

ALTER TABLE organization_members
  ADD CONSTRAINT organization_members_owner_link_check
  CHECK (
    (role = 'owner' AND ended_at IS NULL) = (owner_person_id IS NOT NULL)
    AND (owner_person_id IS NULL OR owner_person_id = person_id)
  );

ALTER TABLE organization_members
  ADD CONSTRAINT organization_members_owner_account_fkey
  FOREIGN KEY (owner_person_id) REFERENCES accounts(person_id) ON DELETE RESTRICT;

CREATE INDEX organization_members_owner_person_idx
  ON organization_members(owner_person_id)
  WHERE owner_person_id IS NOT NULL;
