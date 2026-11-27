CREATE TABLE organization_members (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id bigint NOT NULL REFERENCES organizations(party_id),
    person_id bigint NOT NULL REFERENCES persons(party_id),
    role text NOT NULL CHECK (role IN ('owner', 'member')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    ended_at timestamptz,
    owner_person_id bigint,
    CONSTRAINT organization_members_owner_link_check CHECK (
        (role = 'owner' AND ended_at IS NULL) = (owner_person_id IS NOT NULL)
        AND (owner_person_id IS NULL OR owner_person_id = person_id)
    ),
    CONSTRAINT organization_members_owner_account_fkey
        FOREIGN KEY (owner_person_id) REFERENCES accounts(person_id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX organization_members_one_open_uq
    ON organization_members (organization_id, person_id) WHERE ended_at IS NULL;

CREATE INDEX organization_members_person_idx
    ON organization_members (person_id);

CREATE INDEX organization_members_owner_person_idx
    ON organization_members(owner_person_id) WHERE owner_person_id IS NOT NULL;
