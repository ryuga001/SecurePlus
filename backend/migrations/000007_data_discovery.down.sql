DROP TABLE IF EXISTS data_discovery_policy_rule_mapping;
DROP TABLE IF EXISTS data_discovery_policies;
DROP TABLE IF EXISTS data_discovery_credentials;
DROP TABLE IF EXISTS data_discovery_configurations;
DROP TABLE IF EXISTS data_discovery_source_capabilities;

DELETE FROM role_privileges
WHERE privilege_id IN (SELECT id FROM privileges WHERE name LIKE 'admin.discovery.%');

DELETE FROM privileges WHERE name LIKE 'admin.discovery.%';
