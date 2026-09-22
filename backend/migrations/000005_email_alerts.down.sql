DROP TABLE IF EXISTS alert_policy_mapping;
DROP TABLE IF EXISTS alerts;

DELETE FROM role_privileges
WHERE privilege_id IN (SELECT id FROM privileges WHERE name LIKE 'admin.email.alert.%');

DELETE FROM privileges WHERE name LIKE 'admin.email.alert.%';
