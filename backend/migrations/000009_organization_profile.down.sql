DELETE FROM role_privileges
WHERE privilege_id IN (SELECT id FROM privileges WHERE name = 'admin.organization.edit');

DELETE FROM privileges WHERE name = 'admin.organization.edit';
