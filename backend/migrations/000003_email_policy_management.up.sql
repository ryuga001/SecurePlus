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

CREATE TABLE file_types (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    extension  VARCHAR(20)  NOT NULL UNIQUE CHECK (extension = lower(extension)),
    label      VARCHAR(50)  NOT NULL,
    active     BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE policies (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    policy_name VARCHAR(100) NOT NULL CHECK (char_length(policy_name) BETWEEN 2 AND 100),
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('EMAIL')),
    action      VARCHAR(20)  NOT NULL DEFAULT 'AUDIT'
                CHECK (action IN ('BLOCK', 'AUDIT', 'QUARANTINE', 'REDACT')),
    active      BOOLEAN      NOT NULL DEFAULT true,
    domain_restriction     JSONB NOT NULL DEFAULT '{"mode": "NONE", "values": []}'::jsonb
                CHECK (
                    domain_restriction->>'mode' IN ('NONE', 'BLOCK', 'ALLOW')
                    AND jsonb_typeof(domain_restriction->'values') = 'array'
                    AND (domain_restriction->>'mode' <> 'NONE'
                         OR jsonb_array_length(domain_restriction->'values') = 0)
                ),
    attachment_restriction JSONB NOT NULL DEFAULT '{"mode": "NONE", "values": []}'::jsonb
                CHECK (
                    attachment_restriction->>'mode' IN ('NONE', 'BLOCK', 'ALLOW')
                    AND jsonb_typeof(attachment_restriction->'values') = 'array'
                    AND (attachment_restriction->>'mode' <> 'NONE'
                         OR jsonb_array_length(attachment_restriction->'values') = 0)
                ),
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

INSERT INTO file_types (extension, label) VALUES
('txt',  'Plain text'),
('csv',  'CSV spreadsheet'),
('pdf',  'PDF document'),
('doc',  'Word document (legacy)'),
('docx', 'Word document'),
('xls',  'Excel spreadsheet (legacy)'),
('xlsx', 'Excel spreadsheet'),
('ppt',  'PowerPoint deck (legacy)'),
('pptx', 'PowerPoint deck'),
('zip',  'ZIP archive'),
('rar',  'RAR archive'),
('7z',   '7-Zip archive'),
('png',  'PNG image'),
('jpg',  'JPEG image'),
('jpeg', 'JPEG image'),
('gif',  'GIF image'),
('json', 'JSON file'),
('xml',  'XML file'),
('exe',  'Windows executable'),
('js',   'JavaScript file')
ON CONFLICT (extension) DO NOTHING;

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
('admin.rule.delete',        'DASHBOARD'),
('admin.email.audit.view',   'DASHBOARD'),
('admin.email.incident.view','DASHBOARD')
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
      OR p.name LIKE 'admin.email.audit.%'
      OR p.name LIKE 'admin.email.incident.%'
  )
ON CONFLICT DO NOTHING;

INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'policy_block_notice',
    'Your message was not delivered - {{org_name}}',
    '{"org_name": "Organisation name", "policy_name": "Policies that were triggered", "policy_count": "How many policies were triggered", "reason": "Why the message was blocked", "subject": "Subject of the blocked message", "recipients": "Recipients the message was not delivered to", "message_id": "Message-ID of the blocked message", "correlation_id": "Processing correlation id", "blocked_at": "When the message was blocked"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border:1px solid #e4e7ec;">
          <tr>
            <td style="background-color:#1d4ed8;padding:20px 32px;">
              <p style="margin:0;color:#ffffff;font-size:18px;font-weight:bold;">{{org_name}}</p>
              <p style="margin:4px 0 0;color:#c7d7fe;font-size:13px;">Email protection</p>
            </td>
          </tr>
          <tr>
            <td style="padding:32px;">
              <p style="margin:0 0 16px;color:#101828;font-size:20px;font-weight:bold;">Your message was not delivered</p>
              <p style="margin:0 0 20px;color:#475467;font-size:14px;line-height:22px;">
                A message you sent was stopped by {{org_name}} because it triggered {{reason}}.
                It was not delivered to the recipients listed below.
              </p>
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e4e7ec;font-size:13px;color:#101828;">
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;width:150px;color:#475467;">Subject</td>
                  <td style="padding:10px 14px;">{{subject}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Not delivered to</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{recipients}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Policy</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{policy_name}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Blocked at</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{blocked_at}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Reference</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;font-family:monospace;font-size:12px;">{{correlation_id}}</td>
                </tr>
              </table>
              <p style="margin:20px 0 0;color:#475467;font-size:14px;line-height:22px;">
                If you believe this message should have been allowed, contact your IT or security team
                and quote the reference above.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 32px;background-color:#f9fafb;border-top:1px solid #e4e7ec;">
              <p style="margin:0;color:#98a2b3;font-size:12px;">
                This is an automated message from {{org_name}}. Do not reply to it.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
)
ON CONFLICT (customer_id, name) DO NOTHING;
