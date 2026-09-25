CREATE TABLE data_discovery_scans (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id       INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    policy_id         INT NOT NULL,
    requested_by      INT,
    status            VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'RUNNING', 'PARTIAL', 'COMPLETED', 'FAILED')),
    error_code        VARCHAR(50),
    total_targets     INT    NOT NULL DEFAULT 0 CHECK (total_targets >= 0),
    completed_targets INT    NOT NULL DEFAULT 0 CHECK (completed_targets >= 0),
    failed_targets    INT    NOT NULL DEFAULT 0 CHECK (failed_targets >= 0),
    files_discovered  BIGINT NOT NULL DEFAULT 0 CHECK (files_discovered >= 0),
    files_supported   BIGINT NOT NULL DEFAULT 0 CHECK (files_supported >= 0),
    files_skipped     BIGINT NOT NULL DEFAULT 0 CHECK (files_skipped >= 0),
    files_processed   BIGINT NOT NULL DEFAULT 0 CHECK (files_processed >= 0),
    files_succeeded   BIGINT NOT NULL DEFAULT 0 CHECK (files_succeeded >= 0),
    files_failed      BIGINT NOT NULL DEFAULT 0 CHECK (files_failed >= 0),
    findings_total    BIGINT NOT NULL DEFAULT 0 CHECK (findings_total >= 0),
    bytes_processed   BIGINT NOT NULL DEFAULT 0 CHECK (bytes_processed >= 0),
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, customer_id),
    FOREIGN KEY (policy_id, customer_id)
        REFERENCES data_discovery_policies(id, customer_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX data_discovery_scans_one_active
    ON data_discovery_scans (policy_id)
    WHERE status IN ('PENDING', 'RUNNING');

CREATE INDEX data_discovery_scans_pending
    ON data_discovery_scans (id)
    WHERE status = 'PENDING' AND started_at IS NULL;

CREATE INDEX data_discovery_scans_customer_created
    ON data_discovery_scans (customer_id, created_at DESC);

CREATE TABLE data_discovery_scan_targets (
    scan_id          BIGINT NOT NULL,
    position         INT    NOT NULL CHECK (position >= 0),
    customer_id      INT    NOT NULL,
    target           VARCHAR(1024) NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'PARTIAL', 'FAILED')),
    error_code       VARCHAR(50),
    files_discovered BIGINT NOT NULL DEFAULT 0,
    files_skipped    BIGINT NOT NULL DEFAULT 0,
    files_succeeded  BIGINT NOT NULL DEFAULT 0,
    files_failed     BIGINT NOT NULL DEFAULT 0,
    findings_total   BIGINT NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    PRIMARY KEY (scan_id, position),
    FOREIGN KEY (scan_id, customer_id)
        REFERENCES data_discovery_scans(id, customer_id) ON DELETE CASCADE
);

CREATE TABLE data_discovery_file_results (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scan_id         BIGINT NOT NULL,
    customer_id     INT    NOT NULL,
    target_position INT    NOT NULL,
    file_key        TEXT   NOT NULL CHECK (octet_length(file_key) BETWEEN 1 AND 2048),
    file_name       VARCHAR(1024) NOT NULL,
    extension       VARCHAR(20)   NOT NULL DEFAULT '',
    mime_type       VARCHAR(255)  NOT NULL DEFAULT '',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    modified_at     TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL CHECK (status IN ('SUCCEEDED', 'FAILED')),
    error_code      VARCHAR(50),
    findings_total  BIGINT NOT NULL DEFAULT 0 CHECK (findings_total >= 0),
    findings        JSONB  NOT NULL DEFAULT '[]'::jsonb
                    CHECK (jsonb_typeof(findings) = 'array' AND octet_length(findings::text) <= 65536),
    bytes_processed BIGINT NOT NULL DEFAULT 0,
    duration_ms     BIGINT NOT NULL DEFAULT 0,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scan_id, file_key),
    FOREIGN KEY (scan_id, customer_id)
        REFERENCES data_discovery_scans(id, customer_id) ON DELETE CASCADE,
    FOREIGN KEY (scan_id, target_position)
        REFERENCES data_discovery_scan_targets(scan_id, position) ON DELETE CASCADE
);

CREATE INDEX data_discovery_file_results_findings
    ON data_discovery_file_results (scan_id, findings_total DESC, id);

INSERT INTO privileges (name, type) VALUES
('admin.discovery.scan.view', 'DASHBOARD'),
('admin.discovery.scan.create', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name LIKE 'admin.discovery.scan.%'
ON CONFLICT DO NOTHING;
