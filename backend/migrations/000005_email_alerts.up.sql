CREATE TABLE alerts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id       INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name              CITEXT      NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
    schedule_type     VARCHAR(20) NOT NULL DEFAULT 'REAL_TIME'   CHECK (schedule_type IN ('REAL_TIME', 'CUSTOM')),
    notification_type VARCHAR(20) NOT NULL DEFAULT 'EMAIL'       CHECK (notification_type IN ('EMAIL', 'SMS')),
    target            JSONB       NOT NULL DEFAULT '[]'::jsonb
                      CHECK (
                          jsonb_typeof(target) = 'array'
                          AND jsonb_array_length(target) <= 200
                      ),
    alert_type        VARCHAR(20) NOT NULL DEFAULT 'APPLICATION' CHECK (alert_type IN ('SYSTEM', 'APPLICATION')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name),
    UNIQUE (id, customer_id)
);

CREATE TABLE alert_policy_mapping (
    alert_id    UUID NOT NULL,
    policy_id   INT  NOT NULL,
    customer_id INT  NOT NULL,
    PRIMARY KEY (alert_id, policy_id),
    FOREIGN KEY (alert_id, customer_id)  REFERENCES alerts(id, customer_id)   ON DELETE CASCADE,
    FOREIGN KEY (policy_id, customer_id) REFERENCES policies(id, customer_id) ON DELETE CASCADE
);

INSERT INTO privileges (name, type) VALUES
('admin.email.alert.view', 'DASHBOARD'),
('admin.email.alert.create', 'DASHBOARD'),
('admin.email.alert.edit', 'DASHBOARD'),
('admin.email.alert.delete', 'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND p.name LIKE 'admin.email.alert.%'
ON CONFLICT DO NOTHING;
