CREATE TABLE parties (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

COMMENT ON TABLE parties IS 'BDSPro canonical bootstrap v1: shared party identity.';
