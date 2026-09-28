ALTER TABLE roles
    ADD COLUMN description VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ADD COLUMN updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now();

ALTER TABLE dashboard_users DROP CONSTRAINT dashboard_users_role_id_fkey;

ALTER TABLE dashboard_users ADD CONSTRAINT dashboard_users_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles(id);
