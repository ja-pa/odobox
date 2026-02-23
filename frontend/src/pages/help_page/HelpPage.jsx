function HelpPage() {
  return (
    <section>
      <header className="section-header">
        <h2>Help</h2>
        <p>Quick setup for Odorik voicemail and API access used by OdoBox.</p>
      </header>

      <div className="settings-layout help-layout">
        <section className="settings-card">
          <h3>1) Enable Voicemail Emails in Odorik</h3>
          <ol className="help-steps">
            <li>Log in to Odorik web portal.</li>
            <li>Open voicemail/call settings for your line.</li>
            <li>Enable sending voicemail recordings by email.</li>
            <li>Set destination mailbox that OdoBox reads (for example Gmail IMAP account).</li>
            <li>Make a test call and leave a message.</li>
          </ol>
          <p className="help-note">
            In OdoBox, run <strong>Sync</strong> and verify that the message appears in Inbox.
          </p>
        </section>

        <section className="settings-card">
          <h3>2) Configure IMAP in OdoBox Settings</h3>
          <ul className="help-list">
            <li>
              <strong>Host/Port:</strong> for Gmail use <code>imap.gmail.com</code> and{' '}
              <code>993</code>.
            </li>
            <li>
              <strong>SSL/TLS:</strong> keep enabled.
            </li>
            <li>
              <strong>Username:</strong> full mailbox login (usually full email).
            </li>
            <li>
              <strong>Password:</strong> mailbox password or app password (recommended for Gmail).
            </li>
            <li>
              <strong>Mailbox folder:</strong> typically <code>INBOX</code>.
            </li>
          </ul>
        </section>

        <section className="settings-card">
          <h3>3) Configure Odorik API Credentials</h3>
          <ul className="help-list">
            <li>
              In Odorik account, create/find API credentials (user + API password/PIN).
            </li>
            <li>
              In <strong>Settings → Odorik API Access</strong> fill:
            </li>
            <li>
              <strong>Odorik API User</strong> = your Odorik numeric login.
            </li>
            <li>
              <strong>Odorik API Password</strong> = API password/PIN used for SMS and balance.
            </li>
            <li>
              <strong>Legacy PIN</strong> = optional fallback for older endpoints.
            </li>
            <li>
              <strong>Default Sender ID</strong> = optional default sender.
            </li>
          </ul>
          <p className="help-note">
            After save, open SMS page and send one test SMS to verify credentials.
          </p>
        </section>

        <section className="settings-card">
          <h3>4) Common Checks</h3>
          <ul className="help-list">
            <li>
              If Inbox is empty, verify voicemail emails really arrive in the configured mailbox.
            </li>
            <li>If balance shows unavailable, check Odorik user/password in Settings.</li>
            <li>
              If sender shows only SMSinfo, set <strong>Default SMS identity text</strong> so
              recipients know who sent it.
            </li>
          </ul>
        </section>
      </div>
    </section>
  )
}

export default HelpPage
