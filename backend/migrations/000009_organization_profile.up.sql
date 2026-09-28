INSERT INTO privileges (name, type) VALUES
('admin.organization.edit', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name = 'admin.organization.edit'
ON CONFLICT DO NOTHING;
