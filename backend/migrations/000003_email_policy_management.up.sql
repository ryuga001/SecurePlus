CREATE TABLE groups (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('USER')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name),
    UNIQUE (id, customer_id, type)
);

CREATE TABLE email_users (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    email       CITEXT       NOT NULL UNIQUE CHECK (char_length(email) BETWEEN 3 AND 100),
    first_name  VARCHAR(50)  NOT NULL,
    last_name   VARCHAR(50)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (id, customer_id)
);

CREATE TABLE policies (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    policy_name VARCHAR(100) NOT NULL CHECK (char_length(policy_name) BETWEEN 2 AND 100),
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('EMAIL')),
    active      BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, policy_name),
    UNIQUE (id, customer_id)
);

CREATE TABLE rules (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    rule_name   VARCHAR(100) NOT NULL CHECK (char_length(rule_name) BETWEEN 2 AND 100),
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('REGEX', 'KEYWORD')),
    value       TEXT         NOT NULL CHECK (char_length(value) BETWEEN 1 AND 1000),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, rule_name),
    UNIQUE (id, customer_id)
);

CREATE TABLE email_user_group_mapping (
    email_user_id INT NOT NULL,
    group_id      INT NOT NULL,
    customer_id   INT NOT NULL,
    group_type    VARCHAR(20) NOT NULL DEFAULT 'USER' CHECK (group_type = 'USER'),
    PRIMARY KEY (email_user_id, group_id),
    FOREIGN KEY (email_user_id, customer_id) REFERENCES email_users(id, customer_id) ON DELETE CASCADE,
    FOREIGN KEY (group_id, customer_id, group_type) REFERENCES groups(id, customer_id, type) ON DELETE CASCADE
);

CREATE TABLE policy_group_mapping (
    policy_id   INT NOT NULL,
    group_id    INT NOT NULL,
    customer_id INT NOT NULL,
    group_type  VARCHAR(20) NOT NULL DEFAULT 'USER' CHECK (group_type = 'USER'),
    PRIMARY KEY (policy_id, group_id),
    FOREIGN KEY (policy_id, customer_id) REFERENCES policies(id, customer_id) ON DELETE CASCADE,
    FOREIGN KEY (group_id, customer_id, group_type) REFERENCES groups(id, customer_id, type) ON DELETE CASCADE
);

CREATE TABLE policy_rule_mapping (
    policy_id   INT NOT NULL,
    rule_id     INT NOT NULL,
    customer_id INT NOT NULL,
    PRIMARY KEY (policy_id, rule_id),
    FOREIGN KEY (policy_id, customer_id) REFERENCES policies(id, customer_id) ON DELETE CASCADE,
    FOREIGN KEY (rule_id, customer_id) REFERENCES rules(id, customer_id) ON DELETE CASCADE
);

INSERT INTO privileges (name, type) VALUES
('admin.policy.view',        'DASHBOARD'),
('admin.policy.create',      'DASHBOARD'),
('admin.policy.edit',        'DASHBOARD'),
('admin.policy.delete',      'DASHBOARD'),
('admin.email.user.view',    'DASHBOARD'),
('admin.email.user.create',  'DASHBOARD'),
('admin.email.user.edit',    'DASHBOARD'),
('admin.email.user.delete',  'DASHBOARD'),
('admin.email.group.view',   'DASHBOARD'),
('admin.email.group.create', 'DASHBOARD'),
('admin.email.group.edit',   'DASHBOARD'),
('admin.email.group.delete', 'DASHBOARD'),
('admin.rule.view',          'DASHBOARD'),
('admin.rule.create',        'DASHBOARD'),
('admin.rule.edit',          'DASHBOARD'),
('admin.rule.delete',        'DASHBOARD')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN privileges p
WHERE r.type = 'admin'
  AND (
      p.name LIKE 'admin.policy.%'
      OR p.name LIKE 'admin.email.user.%'
      OR p.name LIKE 'admin.email.group.%'
      OR p.name LIKE 'admin.rule.%'
  )
ON CONFLICT DO NOTHING;
