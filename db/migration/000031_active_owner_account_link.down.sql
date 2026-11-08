-- Development-only rollback. This drops the database-level account guard;
-- it must never be used to bypass account/owner lifecycle review.
DROP INDEX organization_members_owner_person_idx;
ALTER TABLE organization_members DROP CONSTRAINT organization_members_owner_account_fkey;
ALTER TABLE organization_members DROP CONSTRAINT organization_members_owner_link_check;
ALTER TABLE organization_members DROP COLUMN owner_person_id;
