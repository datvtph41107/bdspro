-- CORE-01: durable Party identity; exactly one subtype at transaction commit.
BEGIN;
CREATE TABLE parties (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 kind TEXT NOT NULL CHECK (kind IN ('PERSON','ORGANIZATION')),
 UNIQUE (id,kind)
);
CREATE TABLE persons (
 party_id BIGINT PRIMARY KEY,
 kind TEXT NOT NULL DEFAULT 'PERSON' CHECK (kind='PERSON'),
 FOREIGN KEY (party_id,kind) REFERENCES parties(id,kind)
);
CREATE TABLE organizations (
 party_id BIGINT PRIMARY KEY,
 kind TEXT NOT NULL DEFAULT 'ORGANIZATION' CHECK (kind='ORGANIZATION'),
 name TEXT NOT NULL CHECK (btrim(name)<>''),
 tax_code TEXT, phone TEXT, email TEXT, website TEXT, description TEXT,
 archived_at TIMESTAMPTZ,
 FOREIGN KEY (party_id,kind) REFERENCES parties(id,kind)
);
CREATE TABLE person_profiles (
 person_id BIGINT PRIMARY KEY REFERENCES persons(party_id),
 display_name TEXT NOT NULL CHECK(btrim(display_name)<>''),avatar_url TEXT,bio TEXT
);
CREATE FUNCTION assert_party_subtype() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE pid BIGINT; k TEXT; children INTEGER;
BEGIN
 IF TG_TABLE_NAME='parties' THEN
  pid := NEW.id;
 ELSIF TG_OP='DELETE' THEN
  pid := OLD.party_id;
 ELSE
  pid := NEW.party_id;
 END IF;
 -- ON DELETE of Party is intentionally protected by child foreign keys.
 SELECT kind INTO k FROM parties WHERE id=pid;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT (SELECT count(*) FROM persons WHERE party_id=pid)
      + (SELECT count(*) FROM organizations WHERE party_id=pid)
 INTO children;
 IF children <> 1 OR
    (k='PERSON' AND NOT EXISTS(SELECT 1 FROM persons WHERE party_id=pid)) OR
    (k='ORGANIZATION' AND NOT EXISTS(SELECT 1 FROM organizations WHERE party_id=pid))
 THEN RAISE EXCEPTION 'C01: Party % must have exactly one matching subtype',pid USING ERRCODE='23514';
 END IF;
 RETURN NULL;
END $$;
-- Deferred triggers evaluate the FINAL state at COMMIT, not intermediary inserts.
CREATE CONSTRAINT TRIGGER c01_party
 AFTER INSERT OR UPDATE ON parties DEFERRABLE INITIALLY DEFERRED
 FOR EACH ROW EXECUTE FUNCTION assert_party_subtype();
-- Children have party_id in OLD or NEW; UPDATE of party_id forbidden by PK/FK usage.
CREATE CONSTRAINT TRIGGER c01_person
 AFTER INSERT OR UPDATE OR DELETE ON persons DEFERRABLE INITIALLY DEFERRED
 FOR EACH ROW EXECUTE FUNCTION assert_party_subtype();
CREATE CONSTRAINT TRIGGER c01_organization
 AFTER INSERT OR UPDATE OR DELETE ON organizations DEFERRABLE INITIALLY DEFERRED
 FOR EACH ROW EXECUTE FUNCTION assert_party_subtype();
COMMIT;
