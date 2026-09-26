-- Chorus dev demo-data seed (idempotent). Applied by the `seed-dev` CI gate
-- AFTER the backend is healthy so migrations have created the schema. Because
-- demo passwords must be bcrypt-hashed, users are created through the public
-- /api/v1/auth/register endpoint by deploy/ci/seed-dev.sh, NOT raw SQL.
-- This file is intentionally tiny and re-runnable (no-op until the schema
-- exists); it documents the two canonical dev accounts established by the gate.

-- Registered by deploy/ci/seed-dev.sh via the register API:
--   uhsarp@gmail.com     / Demor@cer1  (en -> es, displayName 'Prashanth')
--   avcxafefwer@gmail.com / Demor@cer1 (es -> en, displayName 'avcxafefwer')

-- Sanity pivot so an operator can confirm the seed ran:
--   SELECT email FROM users WHERE email IN ('uhsarp@gmail.com', 'avcxafefwer@gmail.com');
