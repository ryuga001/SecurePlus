DROP TABLE IF EXISTS data_discovery_file_results;
DROP TABLE IF EXISTS data_discovery_scan_targets;
DROP TABLE IF EXISTS data_discovery_scans;

DELETE FROM role_privileges
WHERE privilege_id IN (SELECT id FROM privileges WHERE name LIKE 'admin.discovery.scan.%');

DELETE FROM privileges WHERE name LIKE 'admin.discovery.scan.%';
