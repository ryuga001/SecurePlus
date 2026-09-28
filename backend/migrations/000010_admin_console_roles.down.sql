DROP INDEX IF EXISTS dashboard_users_customer_role_idx;

ALTER TABLE dashboard_users DROP CONSTRAINT dashboard_users_role_id_fkey;

ALTER TABLE dashboard_users ADD CONSTRAINT dashboard_users_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL;

ALTER TABLE roles
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS description;
