import { useEffect, useState } from 'react'
import { DEFAULT_SETTINGS } from '../../settingsApi'
import { exportVCF, importVCF } from '../address_book_page/addressBookApi'

function SettingPage({
  initialSettings = DEFAULT_SETTINGS,
  isLoading = false,
  errorMessage = '',
  onSaveSettings,
  onReloadSettings,
}) {
  const [settings, setSettings] = useState(initialSettings)
  const [saveMessage, setSaveMessage] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [vcfMessage, setVcfMessage] = useState('')
  const [isVcfBusy, setIsVcfBusy] = useState(false)

  useEffect(() => {
    setSettings(initialSettings)
  }, [initialSettings])

  const updateImapField = (field, value) => {
    setSettings((prev) => ({
      ...prev,
      imap: {
        ...prev.imap,
        [field]: value,
      },
    }))
  }

  const updateMessageCleanerField = (field, value) => {
    setSettings((prev) => ({
      ...prev,
      messageCleaner: {
        ...prev.messageCleaner,
        [field]: value,
      },
    }))
  }

  const updateVoicemailParserField = (field, value) => {
    setSettings((prev) => ({
      ...prev,
      voicemailParser: {
        ...prev.voicemailParser,
        [field]: value,
      },
    }))
  }

  const updateSMSParserField = (field, value) => {
    setSettings((prev) => ({
      ...prev,
      smsParser: {
        ...prev.smsParser,
        [field]: value,
      },
    }))
  }

  const handleSave = async (event) => {
    event.preventDefault()
    setIsSaving(true)
    setSaveMessage('')
    try {
      await onSaveSettings?.(settings)
      setSaveMessage(`Saved at ${new Date().toLocaleTimeString()}`)
    } catch (error) {
      setSaveMessage(`Save failed: ${error.message}`)
    } finally {
      setIsSaving(false)
    }
  }

  const handleReload = async () => {
    setSaveMessage('')
    try {
      await onReloadSettings?.()
      setSaveMessage('Settings reloaded from API.')
    } catch (error) {
      setSaveMessage(`Reload failed: ${error.message}`)
    }
  }

  const handleImportVcf = async (event) => {
    const file = event.target.files?.[0]
    if (!file) return
    setVcfMessage('')
    setIsVcfBusy(true)
    try {
      const content = await file.text()
      const result = await importVCF(content)
      setVcfMessage(
        `VCF imported. Processed: ${result.processed}, inserted: ${result.imported}, updated: ${result.updated}, skipped: ${result.skipped}`
      )
    } catch (error) {
      setVcfMessage(`VCF import failed: ${error.message}`)
    } finally {
      setIsVcfBusy(false)
      event.target.value = ''
    }
  }

  const handleExportVcf = async () => {
    setVcfMessage('')
    setIsVcfBusy(true)
    try {
      const result = await exportVCF()
      const blob = new Blob([result.content ?? ''], { type: 'text/vcard;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'contacts.vcf'
      a.click()
      URL.revokeObjectURL(url)
      setVcfMessage(`VCF exported (${result.count} contacts).`)
    } catch (error) {
      setVcfMessage(`VCF export failed: ${error.message}`)
    } finally {
      setIsVcfBusy(false)
    }
  }

  return (
    <section>
      <header className="section-header">
        <h2>Settings</h2>
        <p>Configure API-backed IMAP, transcript cleaning and parser behavior.</p>
      </header>

      <form className="settings-layout" onSubmit={handleSave}>
        <section className="settings-card">
          <h3>IMAP Settings</h3>
          <div className="settings-grid">
            <label className="form-field">
              <span>Host</span>
              <input
                type="text"
                placeholder="imap.example.com"
                value={settings.imap.host}
                onChange={(event) => updateImapField('host', event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="form-field">
              <span>Port</span>
              <input
                type="number"
                min="1"
                max="65535"
                value={settings.imap.port}
                onChange={(event) => updateImapField('port', event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="form-field">
              <span>Username</span>
              <input
                type="text"
                value={settings.imap.username}
                onChange={(event) => updateImapField('username', event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="form-field">
              <span>Password</span>
              <input
                type="password"
                value={settings.imap.password}
                onChange={(event) => updateImapField('password', event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="form-field">
              <span>Mailbox folder</span>
              <input
                type="text"
                value={settings.imap.mailbox}
                onChange={(event) => updateImapField('mailbox', event.target.value)}
                disabled={isLoading || isSaving}
              />
            </label>
            <label className="checkbox-field">
              <input
                type="checkbox"
                checked={settings.imap.secure}
                onChange={(event) => updateImapField('secure', event.target.checked)}
                disabled={isLoading || isSaving}
              />
              <span>Use SSL/TLS</span>
            </label>
          </div>
        </section>

        <section className="settings-card">
          <h3>Odorik API Access</h3>
          <label className="form-field">
            <span>Odorik API User</span>
            <input
              type="text"
              placeholder="e.g. 7089454"
              value={settings.odorik.user}
              onChange={(event) =>
                setSettings((prev) => ({
                  ...prev,
                  odorik: { ...prev.odorik, user: event.target.value },
                }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
          <label className="form-field">
            <span>Odorik API Password</span>
            <input
              type="password"
              placeholder="API password from Odorik"
              value={settings.odorik.password}
              onChange={(event) =>
                setSettings((prev) => ({
                  ...prev,
                  odorik: { ...prev.odorik, password: event.target.value },
                }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
          <label className="form-field">
            <span>Default Sender ID</span>
            <input
              type="text"
              placeholder="Optional sender ID"
              value={settings.odorik.senderId}
              onChange={(event) =>
                setSettings((prev) => ({
                  ...prev,
                  odorik: { ...prev.odorik, senderId: event.target.value },
                }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
          <label className="form-field">
            <span>Legacy PIN (optional)</span>
            <input
              type="password"
              placeholder="Backward compatibility only"
              value={settings.odorik.pin}
              onChange={(event) =>
                setSettings((prev) => ({
                  ...prev,
                  odorik: { ...prev.odorik, pin: event.target.value },
                }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
          <label className="form-field">
            <span>Default SMS identity text</span>
            <input
              type="text"
              placeholder="e.g. Reception Desk"
              value={settings.smsIdentityText ?? ''}
              onChange={(event) =>
                setSettings((prev) => ({
                  ...prev,
                  smsIdentityText: event.target.value,
                }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
        </section>

        <section className="settings-card">
          <h3>Email Check Timer</h3>
          <label className="form-field">
            <span>Check every (minutes)</span>
            <input
              type="number"
              min="1"
              max="1440"
              value={settings.pollIntervalMinutes}
              onChange={(event) =>
                setSettings((prev) => ({ ...prev, pollIntervalMinutes: event.target.value }))
              }
              disabled={isLoading || isSaving}
            />
          </label>
        </section>

        <section className="settings-card">
          <h3>Transcript Version</h3>
          <label className="form-field">
            <span>Default transcript to display</span>
            <select
              value={settings.transcriptVersion}
              onChange={(event) =>
                setSettings((prev) => ({ ...prev, transcriptVersion: event.target.value }))
              }
              disabled={isLoading || isSaving}
            >
              <option value="v1">v1</option>
              <option value="v2">v2</option>
              <option value="both">both</option>
            </select>
          </label>
        </section>

        <section className="settings-card">
          <h3>Message Cleaner</h3>
          <label className="form-field">
            <span>Keep line regex</span>
            <input
              type="text"
              value={settings.messageCleaner.keepLineRegex}
              onChange={(event) => updateMessageCleanerField('keepLineRegex', event.target.value)}
              disabled={isLoading || isSaving}
            />
          </label>
          <label className="form-field">
            <span>Version v1 regex</span>
            <textarea
              value={settings.messageCleaner.versionV1Regex}
              onChange={(event) => updateMessageCleanerField('versionV1Regex', event.target.value)}
              disabled={isLoading || isSaving}
              rows={3}
            />
          </label>
          <label className="form-field">
            <span>Version v2 regex</span>
            <textarea
              value={settings.messageCleaner.versionV2Regex}
              onChange={(event) => updateMessageCleanerField('versionV2Regex', event.target.value)}
              disabled={isLoading || isSaving}
              rows={3}
            />
          </label>
          <label className="form-field">
            <span>Remove regexes (one per line)</span>
            <textarea
              value={settings.messageCleaner.removeRegexes}
              onChange={(event) => updateMessageCleanerField('removeRegexes', event.target.value)}
              disabled={isLoading || isSaving}
              rows={5}
            />
          </label>
          <label className="checkbox-field">
            <input
              type="checkbox"
              checked={settings.messageCleaner.collapseBlankLines}
              onChange={(event) =>
                updateMessageCleanerField('collapseBlankLines', event.target.checked)
              }
              disabled={isLoading || isSaving}
            />
            <span>Collapse blank lines</span>
          </label>
        </section>

        <section className="settings-card">
          <h3>Voicemail Parser</h3>
          <label className="form-field">
            <span>Caller phone regex</span>
            <textarea
              value={settings.voicemailParser.callerPhoneRegex}
              onChange={(event) => updateVoicemailParserField('callerPhoneRegex', event.target.value)}
              disabled={isLoading || isSaving}
              rows={3}
            />
          </label>
        </section>

        <section className="settings-card">
          <h3>SMS Parser</h3>
          <label className="form-field">
            <span>SMS text extract regex</span>
            <textarea
              value={settings.smsParser.textExtractRegex}
              onChange={(event) => updateSMSParserField('textExtractRegex', event.target.value)}
              disabled={isLoading || isSaving}
              rows={3}
            />
          </label>
        </section>

        <section className="settings-card">
          <h3>Address Book VCF</h3>
          <label className="form-field">
            <span>Import contacts from VCF file</span>
            <input
              type="file"
              accept=".vcf,text/vcard,text/x-vcard"
              onChange={handleImportVcf}
              disabled={isVcfBusy}
            />
          </label>
          <div className="settings-actions">
            <button type="button" className="action-secondary" onClick={handleExportVcf} disabled={isVcfBusy}>
              {isVcfBusy ? 'Working...' : 'Export contacts to VCF'}
            </button>
            {vcfMessage ? <span className="save-message">{vcfMessage}</span> : null}
          </div>
        </section>

        <div className="settings-actions">
          <button type="submit" className="action-primary" disabled={isLoading || isSaving}>
            {isSaving ? 'Saving...' : 'Save settings'}
          </button>
          <button
            type="button"
            className="action-secondary"
            onClick={handleReload}
            disabled={isLoading || isSaving}
          >
            Reload
          </button>
          {errorMessage ? <span className="save-message">{errorMessage}</span> : null}
          {saveMessage ? <span className="save-message">{saveMessage}</span> : null}
        </div>
      </form>
    </section>
  )
}

export default SettingPage
