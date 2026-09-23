CREATE TABLE data_discovery_source_capabilities (
    configuration_type VARCHAR(30) NOT NULL
        CHECK (configuration_type IN (
            'MICROSOFT_ENTRA_ACCOUNT', 'AZURE_STORAGE_ACCOUNT',
            'GOOGLE_SERVICE_ACCOUNT', 'AWS_IAM')),
    source_type        VARCHAR(30) NOT NULL
        CHECK (source_type IN (
            'SHARE_POINT', 'ONE_DRIVE', 'AZURE_BLOB', 'GOOGLE_DRIVE', 'AWS_S3')),
    label              VARCHAR(50)  NOT NULL,
    active             BOOLEAN      NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (configuration_type, source_type)
);

CREATE TABLE data_discovery_configurations (
    id                 INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id        INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name               CITEXT       NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
    description        VARCHAR(255),
    configuration_type VARCHAR(30)  NOT NULL
        CHECK (configuration_type IN (
            'MICROSOFT_ENTRA_ACCOUNT', 'AZURE_STORAGE_ACCOUNT',
            'GOOGLE_SERVICE_ACCOUNT', 'AWS_IAM')),
    config             JSONB        NOT NULL
        CHECK (
            jsonb_typeof(config) = 'object'
            AND COALESCE(
                CASE configuration_type
                    WHEN 'MICROSOFT_ENTRA_ACCOUNT' THEN
                        jsonb_typeof(config->'tenantId') = 'string'
                        AND jsonb_typeof(config->'clientId') = 'string'
                        AND char_length(config->>'tenantId') BETWEEN 1 AND 100
                        AND char_length(config->>'clientId') BETWEEN 1 AND 100
                    WHEN 'AZURE_STORAGE_ACCOUNT' THEN
                        jsonb_typeof(config->'storageAccount') = 'string'
                        AND jsonb_typeof(config->'tenantId') = 'string'
                        AND jsonb_typeof(config->'clientId') = 'string'
                        AND config->>'authMode' = 'SERVICE_PRINCIPAL'
                        AND char_length(config->>'storageAccount') BETWEEN 3 AND 24
                        AND config->>'storageAccount' = lower(config->>'storageAccount')
                    WHEN 'GOOGLE_SERVICE_ACCOUNT' THEN
                        jsonb_typeof(config->'projectId') = 'string'
                        AND jsonb_typeof(config->'clientEmail') = 'string'
                        AND jsonb_typeof(config->'clientId') = 'string'
                        AND jsonb_typeof(config->'tokenUri') = 'string'
                        AND config->>'accessMode' IN ('DOMAIN_WIDE_DELEGATION', 'SHARED_DRIVE')
                        AND (config->>'accessMode' <> 'DOMAIN_WIDE_DELEGATION'
                             OR jsonb_typeof(config->'subject') = 'string')
                    WHEN 'AWS_IAM' THEN
                        jsonb_typeof(config->'accessKeyId') = 'string'
                        AND jsonb_typeof(config->'region') = 'string'
                        AND char_length(config->>'accessKeyId') BETWEEN 16 AND 128
                        AND char_length(config->>'region') BETWEEN 2 AND 30
                    ELSE false
                END, false)
            AND octet_length(config::text) <= 8192
        ),
    status             VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                       CHECK (status IN ('ACTIVE', 'INACTIVE')),
    last_tested_at     TIMESTAMPTZ  NOT NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name),
    UNIQUE (id, customer_id),
    UNIQUE (id, customer_id, configuration_type)
);

CREATE TABLE data_discovery_credentials (
    id               INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id      INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    configuration_id INT NOT NULL,
    secret           BYTEA NOT NULL CHECK (octet_length(secret) BETWEEN 29 AND 65536),
    key_version      INT   NOT NULL DEFAULT 1 CHECK (key_version >= 1),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (configuration_id),
    UNIQUE (id, customer_id),
    FOREIGN KEY (configuration_id, customer_id)
        REFERENCES data_discovery_configurations(id, customer_id) ON DELETE CASCADE
);

CREATE TABLE data_discovery_policies (
    id                 INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id        INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name               CITEXT       NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
    description        VARCHAR(255),
    configuration_id   INT NOT NULL,
    configuration_type VARCHAR(30) NOT NULL,
    source_type        VARCHAR(30) NOT NULL
        CHECK (source_type IN (
            'SHARE_POINT', 'ONE_DRIVE', 'AZURE_BLOB', 'GOOGLE_DRIVE', 'AWS_S3')),
    target_list        JSONB NOT NULL
        CHECK (
            jsonb_typeof(target_list) = 'array'
            AND jsonb_array_length(target_list) BETWEEN 1 AND 200
            AND octet_length(target_list::text) <= 32768
        ),
    file_types         JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (
            jsonb_typeof(file_types) = 'array'
            AND jsonb_array_length(file_types) <= 200
        ),
    status             VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                       CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name),
    UNIQUE (id, customer_id),
    FOREIGN KEY (configuration_id, customer_id, configuration_type)
        REFERENCES data_discovery_configurations(id, customer_id, configuration_type)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (configuration_type, source_type)
        REFERENCES data_discovery_source_capabilities(configuration_type, source_type)
        ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE data_discovery_policy_rule_mapping (
    policy_id   INT NOT NULL,
    rule_id     INT NOT NULL,
    customer_id INT NOT NULL,
    PRIMARY KEY (policy_id, rule_id),
    FOREIGN KEY (policy_id, customer_id)
        REFERENCES data_discovery_policies(id, customer_id) ON DELETE CASCADE,
    FOREIGN KEY (rule_id, customer_id)
        REFERENCES rules(id, customer_id) ON DELETE CASCADE
);

INSERT INTO data_discovery_source_capabilities (configuration_type, source_type, label) VALUES
('MICROSOFT_ENTRA_ACCOUNT', 'SHARE_POINT', 'SharePoint'),
('MICROSOFT_ENTRA_ACCOUNT', 'ONE_DRIVE', 'OneDrive'),
('AZURE_STORAGE_ACCOUNT', 'AZURE_BLOB', 'Azure Blob Storage'),
('GOOGLE_SERVICE_ACCOUNT', 'GOOGLE_DRIVE', 'Google Drive'),
('AWS_IAM', 'AWS_S3', 'Amazon S3')
ON CONFLICT (configuration_type, source_type) DO NOTHING;

INSERT INTO privileges (name, type) VALUES
('admin.discovery.configuration.view', 'DASHBOARD'),
('admin.discovery.configuration.create', 'DASHBOARD'),
('admin.discovery.configuration.edit', 'DASHBOARD'),
('admin.discovery.configuration.delete', 'DASHBOARD'),
('admin.discovery.configuration.test', 'DASHBOARD'),
('admin.discovery.policy.view', 'DASHBOARD'),
('admin.discovery.policy.create', 'DASHBOARD'),
('admin.discovery.policy.edit', 'DASHBOARD'),
('admin.discovery.policy.delete', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name LIKE 'admin.discovery.%'
ON CONFLICT DO NOTHING;
