CREATE TABLE customer_branding (
    customer_id INT PRIMARY KEY REFERENCES customers(id) ON DELETE CASCADE,
    logo_key    VARCHAR(512),
    theme       VARCHAR(10) NOT NULL DEFAULT 'LIGHT'   CHECK (theme IN ('LIGHT', 'DARK')),
    language    VARCHAR(20) NOT NULL DEFAULT 'ENGLISH' CHECK (language IN ('ENGLISH', 'JAPANESE', 'SPANISH')),
    timezone    VARCHAR(64) NOT NULL DEFAULT 'UTC',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO customer_branding (customer_id)
SELECT id FROM customers
ON CONFLICT (customer_id) DO NOTHING;

INSERT INTO privileges (name, type) VALUES
('admin.branding.edit', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name = 'admin.branding.edit'
ON CONFLICT DO NOTHING;
