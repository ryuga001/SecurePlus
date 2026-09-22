INSERT INTO email_templates (customer_id, name, subject, variables, body) VALUES
(
    1,
    'policy_breach_alert',
    'Policy breach detected - {{org_name}}',
    '{"org_name": "Organisation name", "alert_name": "Name of the alert that fired", "sender": "Sender of the offending message", "recipients": "Intended recipients", "subject": "Subject of the offending message", "decision": "PASS or FLAGGED", "trigger": "RESTRICTION or CONTENT", "action": "Effective action taken", "policy_name": "Policies that were breached", "policy_count": "How many policies were breached", "match_count": "Number of content rule matches", "violation_count": "Number of restriction violations", "withheld_count": "Number of recipients withheld", "message_id": "Message-ID of the offending message", "correlation_id": "Processing correlation id", "detected_at": "When the breach was detected", "incident_url": "Dashboard link to the incident"}'::jsonb,
    '<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background-color:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f5f7;padding:32px 0;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background-color:#ffffff;border:1px solid #e4e7ec;">
          <tr>
            <td style="background-color:#b42318;padding:20px 32px;">
              <p style="margin:0;color:#ffffff;font-size:18px;font-weight:bold;">{{org_name}}</p>
              <p style="margin:4px 0 0;color:#fecdca;font-size:13px;">Policy breach alert</p>
            </td>
          </tr>
          <tr>
            <td style="padding:32px;">
              <p style="margin:0 0 8px;color:#101828;font-size:20px;font-weight:bold;">A policy breach was detected</p>
              <p style="margin:0 0 20px;color:#475467;font-size:14px;line-height:22px;">
                Alert <strong>{{alert_name}}</strong> fired because a message breached {{policy_count}} policy
                configuration(s) in your organisation. Open the incident for the full rule matches, restriction
                violations and withheld recipients.
              </p>

              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e4e7ec;font-size:13px;color:#101828;">
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;width:170px;color:#475467;">Sender</td>
                  <td style="padding:10px 14px;">{{sender}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Recipients</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{recipients}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Subject</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{subject}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Policies breached</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{policy_name}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Trigger</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{trigger}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Decision / action</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{decision}} / {{action}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Findings</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">
                    {{match_count}} rule match(es), {{violation_count}} restriction violation(s), {{withheld_count}} withheld recipient(s)
                  </td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Detected at</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{detected_at}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Message-ID</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{message_id}}</td>
                </tr>
                <tr>
                  <td style="padding:10px 14px;background-color:#f9fafb;color:#475467;border-top:1px solid #e4e7ec;">Correlation ID</td>
                  <td style="padding:10px 14px;border-top:1px solid #e4e7ec;">{{correlation_id}}</td>
                </tr>
              </table>

              <p style="margin:24px 0 0;">
                <a href="{{incident_url}}" style="display:inline-block;padding:11px 18px;background-color:#1d4ed8;color:#ffffff;font-size:14px;text-decoration:none;">View the incident</a>
              </p>
              <p style="margin:20px 0 0;color:#667085;font-size:12px;line-height:18px;">
                You are receiving this because alert {{alert_name}} is configured for real-time email notification.
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
