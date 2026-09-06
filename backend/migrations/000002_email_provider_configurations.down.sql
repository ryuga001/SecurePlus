DROP TABLE IF EXISTS email_provider_configurations;

DELETE FROM privileges WHERE name LIKE 'admin.email.provider.%';
