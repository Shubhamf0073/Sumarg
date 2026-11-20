-- Dev only: a second database for `make test-integration`, same roles. Runs once on first start of the dev volume.
CREATE DATABASE sf_test OWNER sf_owner;
REVOKE ALL ON DATABASE sf_test FROM PUBLIC;
GRANT CONNECT ON DATABASE sf_test TO sf_app, sf_platform;
\c sf_test
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO sf_owner;
