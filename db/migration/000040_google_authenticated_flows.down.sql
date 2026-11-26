DELETE FROM recent_auth_proofs WHERE method='google';
DELETE FROM google_auth_flows WHERE browser_digest IN (SELECT browser_digest FROM google_authenticated_intents);
DROP TABLE google_authenticated_intents;
ALTER TABLE google_auth_flows DROP COLUMN auth_time;
ALTER TABLE recent_auth_proofs DROP CONSTRAINT recent_auth_proofs_method_binding_check;
ALTER TABLE recent_auth_proofs DROP CONSTRAINT recent_auth_proofs_method_check;
ALTER TABLE recent_auth_proofs DROP COLUMN google_subject;
ALTER TABLE recent_auth_proofs ADD CONSTRAINT recent_auth_proofs_method_check CHECK (method IN ('password','verified_email'));
ALTER TABLE recent_auth_proofs ADD CONSTRAINT recent_auth_proofs_check CHECK ((method='verified_email' AND action='password.set' AND email_snapshot<>'') OR (method='password' AND email_snapshot=''));
