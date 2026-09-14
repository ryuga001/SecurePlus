DELETE FROM role_privileges
WHERE privilege_id IN (SELECT id FROM privileges WHERE name = 'admin.branding.edit');

DELETE FROM privileges WHERE name = 'admin.branding.edit';

DROP TABLE IF EXISTS customer_branding;
