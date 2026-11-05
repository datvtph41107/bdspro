CREATE TABLE "persons" (
  "party_id" bigint PRIMARY KEY REFERENCES parties(id)
);
