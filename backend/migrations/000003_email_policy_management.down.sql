DROP TABLE IF EXISTS policy_rule_mapping;
DROP TABLE IF EXISTS policy_group_mapping;
DROP TABLE IF EXISTS email_user_group_mapping;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS email_users;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS file_types;

DELETE FROM privileges
WHERE name LIKE 'admin.policy.%'
   OR name LIKE 'admin.email.user.%'
   OR name LIKE 'admin.email.group.%'
   OR name LIKE 'admin.rule.%';
