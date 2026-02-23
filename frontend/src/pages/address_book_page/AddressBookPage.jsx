import { useEffect, useMemo, useState } from 'react'
import { createContact, deleteContact, listContacts } from './addressBookApi'

const EMPTY_FORM = {
  full_name: '',
  phone: '',
  email: '',
  org: '',
  note: '',
}

function AddressBookPage({ onSendSMSContact }) {
  const [contacts, setContacts] = useState([])
  const [form, setForm] = useState(EMPTY_FORM)
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [searchQuery, setSearchQuery] = useState('')

  const filteredContacts = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) return contacts
    return contacts.filter((contact) => {
      const haystack = [
        contact.full_name || '',
        contact.phone || '',
        contact.email || '',
        contact.org || '',
        contact.note || '',
      ]
        .join(' ')
        .toLowerCase()
      return haystack.includes(query)
    })
  }, [contacts, searchQuery])

  const load = async () => {
    const items = await listContacts()
    setContacts(Array.isArray(items) ? items : [])
  }

  useEffect(() => {
    setIsLoading(true)
    load()
      .catch((error) => setErrorMessage(error.message))
      .finally(() => setIsLoading(false))
  }, [])

  const onSave = async (event) => {
    event.preventDefault()
    setIsSaving(true)
    setStatusMessage('')
    setErrorMessage('')
    try {
      await createContact(form)
      setStatusMessage('Contact created.')
      await load()
      setForm(EMPTY_FORM)
    } catch (error) {
      setErrorMessage(error.message)
    } finally {
      setIsSaving(false)
    }
  }

  const onDelete = async (id) => {
    setStatusMessage('')
    setErrorMessage('')
    try {
      await deleteContact(id)
      await load()
      setStatusMessage('Contact deleted.')
    } catch (error) {
      setErrorMessage(error.message)
    }
  }

  return (
    <section>
      <header className="section-header">
        <h2>Address Book</h2>
        <p>Add contacts manually here. VCF import/export is in Settings.</p>
      </header>

      <div className="settings-layout">
        <section className="settings-card">
          <h3>New Contact</h3>
          <form className="settings-layout" onSubmit={onSave}>
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
                {isSaving ? 'Saving...' : 'Add contact'}
              </button>
            </div>
          </form>
        </section>

        <section className="settings-card">
          <h3>Contacts</h3>
          <label className="form-field">
            <span>Search contacts</span>
            <input
              type="text"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Search by name, phone, email, organization, note"
            />
          </label>
          {isLoading ? (
            <div className="empty-state">Loading contacts...</div>
          ) : contacts.length === 0 ? (
            <div className="empty-state">No contacts yet.</div>
          ) : filteredContacts.length === 0 ? (
            <div className="empty-state">No matching contacts.</div>
          ) : (
            <div className="contact-table">
              {filteredContacts.map((contact) => (
                <article key={contact.id} className="contact-row">
                  <div>
                    <strong>{contact.full_name}</strong>
                    <p>{contact.phone}</p>
                    {contact.email ? <p>{contact.email}</p> : null}
                    {contact.org ? <p>{contact.org}</p> : null}
                    {contact.note ? <p>{contact.note}</p> : null}
                  </div>
                  <div className="contact-row-actions">
                    <button
                      type="button"
                      className="action-secondary send-sms-action"
                      onClick={() => onSendSMSContact?.(contact.phone, contact.full_name)}
                    >
                      ✉ SMS
                    </button>
                    <button type="button" className="action-secondary" onClick={() => onDelete(contact.id)}>
                      Delete
                    </button>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>

        <div className="settings-actions">
          {statusMessage ? <span className="save-message">{statusMessage}</span> : null}
          {errorMessage ? <span className="save-message template-error">{errorMessage}</span> : null}
        </div>
      </div>
    </section>
  )
}

export default AddressBookPage
