import { useEffect, useState } from 'react'
import { t } from '../../../i18n'

const EMPTY_FORM = {
  full_name: '',
  phone: '',
  email: '',
  org: '',
  note: '',
}

function toForm(contact) {
  if (!contact) return EMPTY_FORM
  return {
    full_name: contact.full_name || '',
    phone: contact.phone || '',
    email: contact.email || '',
    org: contact.org || '',
    note: contact.note || '',
  }
}

function ContactModal({
  message,
  isSaving = false,
  errorMessage = '',
  statusMessage = '',
  onClose,
  onSave,
  language = 'en',
}) {
  const contact = message?.contact ?? null
  const [form, setForm] = useState(() => toForm(contact))

  useEffect(() => {
    setForm(toForm(contact))
  }, [contact])

  if (!message || !contact) return null

  const submit = async (event) => {
    event.preventDefault()
    await onSave?.(form)
  }

  return (
    <div className="contact-modal-backdrop" role="presentation" onClick={onClose}>
      <section
        className="contact-modal"
        role="dialog"
        aria-modal="true"
        aria-label="Contact details"
        onClick={(event) => event.stopPropagation()}
      >
        <header className="contact-modal-header">
          <h3>{t(language, 'contact_modal_title')}</h3>
          <button type="button" className="action-secondary" onClick={onClose} disabled={isSaving}>
            {t(language, 'common_close')}
          </button>
        </header>

        <p className="contact-modal-caller">{message.callerPhone || message.caller}</p>

        <form className="settings-layout" onSubmit={submit}>
          <div className="settings-grid">
            <label className="form-field">
              <span>Full name</span>
              <input
                type="text"
                value={form.full_name}
                onChange={(event) => setForm((prev) => ({ ...prev, full_name: event.target.value }))}
                disabled={isSaving}
              />
            </label>
            <label className="form-field">
              <span>Phone</span>
              <input
                type="text"
                value={form.phone}
                onChange={(event) => setForm((prev) => ({ ...prev, phone: event.target.value }))}
                disabled={isSaving}
              />
            </label>
          </div>

          <div className="settings-grid">
            <label className="form-field">
              <span>Email</span>
              <input
                type="text"
                value={form.email}
                onChange={(event) => setForm((prev) => ({ ...prev, email: event.target.value }))}
                disabled={isSaving}
              />
            </label>
            <label className="form-field">
              <span>Organization</span>
              <input
                type="text"
                value={form.org}
                onChange={(event) => setForm((prev) => ({ ...prev, org: event.target.value }))}
                disabled={isSaving}
              />
            </label>
          </div>

          <label className="form-field">
            <span>Note</span>
            <textarea
              rows={3}
              value={form.note}
              onChange={(event) => setForm((prev) => ({ ...prev, note: event.target.value }))}
              disabled={isSaving}
            />
          </label>

          <div className="settings-actions">
            <button type="submit" className="action-primary" disabled={isSaving}>
              {isSaving ? t(language, 'common_saving') : t(language, 'common_save_changes')}
            </button>
            {statusMessage ? <span className="save-message">{statusMessage}</span> : null}
            {errorMessage ? <span className="save-message template-error">{errorMessage}</span> : null}
          </div>
        </form>
      </section>
    </div>
  )
}

export default ContactModal
