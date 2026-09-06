CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE customers (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_name   VARCHAR(100) NOT NULL UNIQUE,
    jwt_secret TEXT         NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(50) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    UNIQUE (customer_id, name)
);

CREATE TABLE privileges (
    id   INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL
);

CREATE TABLE role_privileges (
    role_id      INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    privilege_id INT NOT NULL REFERENCES privileges(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, privilege_id)
);

CREATE TABLE dashboard_users (
    id            INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id   INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    role_id       INT REFERENCES roles(id) ON DELETE SET NULL,
    first_name VARCHAR(50)  NOT NULL,
    last_name  VARCHAR(50)  NOT NULL,
    email         CITEXT       NOT NULL UNIQUE CHECK (char_length(email) <= 100),
    password_hash VARCHAR(255) NOT NULL,
    password_salt VARCHAR(255) NOT NULL DEFAULT gen_random_uuid()::text,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE email_templates (
    id          INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    variables   JSONB        NOT NULL DEFAULT '{}'::jsonb,
    subject     VARCHAR(255) NOT NULL,
    body        TEXT         NOT NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (customer_id, name)
);


INSERT INTO customers (id, org_name)
OVERRIDING SYSTEM VALUE
VALUES (1, 'DPDP Platform')
ON CONFLICT (org_name) DO NOTHING;

SELECT setval(pg_get_serial_sequence('customers', 'id'), (SELECT max(id) FROM customers));


INSERT INTO roles (name, type, customer_id) VALUES
('Super Admin', 'super_admin', 1),
('Admin',       'admin',       1)
ON CONFLICT (customer_id, name) DO NOTHING;

INSERT INTO dashboard_users (customer_id, role_id, email, password_hash, first_name, last_name)
VALUES (
    1,
    (SELECT id FROM roles WHERE customer_id = 1 AND type = 'super_admin'),
    'superadmin@dpdp.local',
    '!',
    'System',
    'Admin'
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'email_verification',
    'Verify your email address',
    '{"name": "Recipient name", "otp": "One-time verification code", "expiry_minutes": "Minutes until the code expires"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Verify your email</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, use the code below to finish creating your account.
              </p>
              <div style="margin:24px 0;padding:16px;background-color:#f3f4f6;border-radius:8px;text-align:center;font-size:28px;font-weight:bold;letter-spacing:6px;color:#111827;">
                {{otp}}
              </div>
              <p style="margin:0;font-size:14px;line-height:1.6;color:#4b5563;">
                This code expires in {{expiry_minutes}} minutes.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
),
(
    1,
    'welcome',
    'Welcome aboard',
    '{"name": "Recipient name", "org_name": "Organization name", "login_url": "Dashboard login URL"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Welcome, {{name}}</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Your {{org_name}} workspace is ready. Sign in to get started.
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0;">
                <tr>
                  <td style="border-radius:8px;background-color:#4f46e5;">
                    <a href="{{login_url}}" style="display:inline-block;padding:12px 28px;color:#ffffff;font-size:15px;font-weight:bold;text-decoration:none;">Go to dashboard</a>
                  </td>
                </tr>
              </table>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                If the button does not work, copy this link into your browser:<br>{{login_url}}
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
),
(
    1,
    'password_reset',
    'Reset your password',
    '{"name": "Recipient name", "otp": "One-time reset code", "expiry_minutes": "Minutes until the code expires"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Password reset</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, enter the code below to reset your password.
              </p>
              <div style="margin:24px 0;padding:16px;background-color:#f3f4f6;border-radius:8px;text-align:center;font-size:28px;font-weight:bold;letter-spacing:6px;color:#111827;">
                {{otp}}
              </div>
              <p style="margin:0 0 8px 0;font-size:14px;line-height:1.6;color:#4b5563;">
                This code expires in {{expiry_minutes}} minutes.
              </p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;">
                If you did not request a password reset, you can ignore this email.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
),
(
    1,
    'user_invited_credentials',
    'Your dashboard account',
    '{"name": "Recipient name", "email": "Login email", "temp_password": "Temporary password", "login_url": "Dashboard login URL"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Account created</h2>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, an account was created for you.
              </p>
              <p style="margin:0 0 8px 0;font-size:15px;color:#1f2937;">Email: <strong>{{email}}</strong></p>
              <p style="margin:0 0 16px 0;font-size:15px;color:#1f2937;">Temporary password: <strong>{{temp_password}}</strong></p>
              <p style="margin:0;font-size:14px;line-height:1.6;color:#4b5563;">
                Sign in at {{login_url}} and change your password immediately.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>'
),
(
    1,
    'password_changed',
    'Your password was changed',
    '{"name": "Recipient name"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:36px 40px 24px 40px;color:#1f2937;">
              <h2 style="margin:0 0 16px 0;font-size:20px;">Password changed</h2>
              <p style="margin:0;font-size:15px;line-height:1.6;color:#4b5563;">
                Hi {{name}}, your password was just changed and all other sessions were signed out.
                If this was not you, reset your password immediately.
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
