CREATE TABLE email_provider_configurations (
    id                      INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id             INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name                    VARCHAR(100) NOT NULL,
    domain                  CITEXT NOT NULL UNIQUE CHECK (char_length(domain) BETWEEN 4 AND 253),
    provider                VARCHAR(20) NOT NULL CHECK (provider IN ('outlook365', 'gmail')),
    dkim_public_key         TEXT,
    dkim_private_key        TEXT,
    access_token_hash       TEXT,
    access_token_expires_at TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO privileges (name, type) VALUES
('admin.email.provider.view',   'DASHBOARD'),
('admin.email.provider.create', 'DASHBOARD'),
('admin.email.provider.edit',   'DASHBOARD'),
('admin.email.provider.delete', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name LIKE 'admin.email.provider.%'
ON CONFLICT DO NOTHING;
