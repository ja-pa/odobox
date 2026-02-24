import { useEffect, useMemo, useState } from 'react'
import {
  createSMSTemplate,
  deleteSMSTemplate,
  getSMSLengthInfo,
  listSMSTemplates,
  sendSMS,
  updateSMSTemplate,
} from './smsApi'
import { listContacts } from '../address_book_page/addressBookApi'
import { t } from '../../i18n'

function composeMessageWithIdentity(identityText, body) {
  const identity = String(identityText ?? '').trim()
  const message = String(body ?? '').trim()
  if (!identity) return message
  if (!message) return identity
  return `${identity}: ${message}`
}

function SmsPage({
  defaultSenderId = '',
  defaultIdentityText = '',
  prefillRecipient = '',
  prefillLabel = '',
  prefillToken = 0,
  language = 'en',
}) {
  const [recipient, setRecipient] = useState('')
  const [sender, setSender] = useState(defaultSenderId)
  const [message, setMessage] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [contacts, setContacts] = useState([])
  const [selectedRecipientContactId, setSelectedRecipientContactId] = useState('')
  const [contactQuery, setContactQuery] = useState('')
  const [showContactPicker, setShowContactPicker] = useState(false)

  const [templates, setTemplates] = useState([])
  const [selectedTemplateId, setSelectedTemplateId] = useState('')
  const [templateName, setTemplateName] = useState('')
  const [templateBody, setTemplateBody] = useState('')
  const [isSavingTemplate, setIsSavingTemplate] = useState(false)
  const [templateMessage, setTemplateMessage] = useState('')
  const [templateErrorMessage, setTemplateErrorMessage] = useState('')
  const [showTemplateManager, setShowTemplateManager] = useState(false)

  const effectiveMessage = useMemo(
    () => composeMessageWithIdentity(defaultIdentityText, message),
    [defaultIdentityText, message]
  )
  const smsInfo = useMemo(() => getSMSLengthInfo(effectiveMessage), [effectiveMessage])
  const filteredContacts = useMemo(() => {
    const query = contactQuery.trim().toLowerCase()
    const base = Array.isArray(contacts) ? contacts : []
    if (!query) return base.slice(0, 12)
    return base
      .filter((contact) => {
        const haystack = [
          contact.full_name || '',
          contact.phone || '',
          contact.email || '',
          contact.org || '',
        ]
          .join(' ')
          .toLowerCase()
        return haystack.includes(query)
      })
      .slice(0, 30)
  }, [contactQuery, contacts])

  const loadTemplates = async () => {
    const items = await listSMSTemplates()
    setTemplates(Array.isArray(items) ? items : [])
  }
  const loadContacts = async () => {
    const items = await listContacts()
    setContacts(Array.isArray(items) ? items : [])
  }

  useEffect(() => {
    loadTemplates().catch((error) => setErrorMessage(error.message))
    loadContacts().catch((error) => setErrorMessage(error.message))
  }, [])

  useEffect(() => {
    if (!prefillToken || !prefillRecipient) return
    setRecipient(prefillRecipient)
    setSelectedRecipientContactId('')
    setShowContactPicker(false)
    setContactQuery(prefillLabel ? `${prefillLabel} (${prefillRecipient})` : prefillRecipient)
  }, [prefillLabel, prefillRecipient, prefillToken])

  const clearTemplateForm = () => {
    setSelectedTemplateId('')
    setTemplateName('')
    setTemplateBody('')
  }

  const applyTemplateToMessage = (templateId) => {
    if (!templateId) return
    const found = templates.find((item) => String(item.id) === String(templateId))
    if (!found) return
    setMessage(found.body || '')
  }

  const onTemplatePick = (value) => {
    setSelectedTemplateId(value)
    if (!value) {
      setTemplateName('')
      setTemplateBody('')
      setTemplateMessage('')
      setMessage('')
      return
    }
    const found = templates.find((item) => String(item.id) === String(value))
    if (!found) return
    setTemplateName(found.name || '')
    setTemplateBody(found.body || '')
    applyTemplateToMessage(value)
  }

  const onSaveTemplate = async () => {
    setTemplateMessage('')
    setTemplateErrorMessage('')
    setErrorMessage('')
    if (!templateName.trim()) {
      setTemplateErrorMessage('Template name is required.')
      return
    }
    if (!templateBody.trim()) {
      setTemplateErrorMessage('Template body is required.')
      return
    }

    setIsSavingTemplate(true)
    try {
      if (selectedTemplateId) {
        const updated = await updateSMSTemplate({
          id: Number(selectedTemplateId),
          name: templateName,
          body: templateBody,
        })
        setTemplateMessage(`Template '${updated.name}' updated.`)
      } else {
        const created = await createSMSTemplate({ name: templateName, body: templateBody })
        setTemplateMessage(`Template '${created.name}' created.`)
        setSelectedTemplateId(String(created.id))
      }
      await loadTemplates()
    } catch (error) {
      setTemplateErrorMessage(error.message)
    } finally {
      setIsSavingTemplate(false)
    }
  }

  const onDeleteTemplate = async () => {
    if (!selectedTemplateId) {
      setErrorMessage('Select a template to delete.')
      return
    }
    setIsSavingTemplate(true)
    setTemplateMessage('')
    setTemplateErrorMessage('')
    setErrorMessage('')
    try {
      await deleteSMSTemplate(Number(selectedTemplateId))
      await loadTemplates()
      clearTemplateForm()
      setTemplateMessage('Template deleted.')
    } catch (error) {
      setTemplateErrorMessage(error.message)
    } finally {
      setIsSavingTemplate(false)
    }
  }

  const onSubmit = async (event) => {
    event.preventDefault()
    setErrorMessage('')
    setStatusMessage('')

    if (!recipient.trim()) {
      setErrorMessage('Recipient is required.')
      return
    }
    if (!message.trim()) {
      setErrorMessage('Message is required.')
      return
    }
    if (!smsInfo.single) {
      setErrorMessage(
        `Message is too long for one SMS (${smsInfo.encoding} ${smsInfo.used}/${smsInfo.max}). Shorten it.`
      )
      return
    }

    setIsSending(true)
    try {
      const result = await sendSMS({ recipient, message, sender })
      setStatusMessage(
        `SMS sent to ${result.recipient} (${result.encoding} ${result.chars_used}/${result.max_single_chars}).`
      )
      setMessage('')
    } catch (error) {
      setErrorMessage(error.message)
    } finally {
      setIsSending(false)
    }
  }

  const onContactPick = (contactId) => {
    setSelectedRecipientContactId(contactId)
    if (!contactId) return
    const found = contacts.find((item) => String(item.id) === String(contactId))
    if (!found) return
    setRecipient(found.phone || '')
    setContactQuery(found.full_name ? `${found.full_name} (${found.phone || ''})` : found.phone || '')
    setShowContactPicker(false)
  }

  const clearPickedContact = () => {
    setSelectedRecipientContactId('')
    setContactQuery('')
  }

  return (
    <section>
      <header className="section-header">
        <h2>{t(language, 'sms_title')}</h2>
        <p>{t(language, 'sms_subtitle')}</p>
      </header>

      <form className="settings-layout" onSubmit={onSubmit}>
        <section className="settings-card">
          <h3>SMS Form</h3>

          <label className="form-field">
            <span>Template (optional)</span>
            <select
              value={selectedTemplateId}
              onChange={(event) => onTemplatePick(event.target.value)}
              disabled={isSending}
            >
              <option value="">No template</option>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
          </label>

          <div className="settings-grid">
            <label className="form-field">
              <span>Recipient number</span>
              <input
                type="text"
                placeholder="+420123456789"
                value={recipient}
                onChange={(event) => {
                  setRecipient(event.target.value)
                  setSelectedRecipientContactId('')
                  setContactQuery('')
                }}
                disabled={isSending}
              />
            </label>
            <label className="form-field">
              <span>Sender (your number/id, optional)</span>
              <input
                type="text"
                placeholder="Leave empty to use default sender from settings"
                value={sender}
                onChange={(event) => setSender(event.target.value)}
                disabled={isSending}
              />
            </label>
          </div>

          <div className="contact-picker">
            <label className="form-field">
              <span>Find recipient in address book</span>
              <input
                type="text"
                placeholder="Search by name, phone, email, company"
                value={contactQuery}
                onFocus={() => setShowContactPicker(true)}
                onChange={(event) => {
                  setContactQuery(event.target.value)
                  setShowContactPicker(true)
                  if (!event.target.value.trim()) setSelectedRecipientContactId('')
                }}
                disabled={isSending}
              />
            </label>

            {selectedRecipientContactId ? (
              <div className="contact-picker-selected">
                <span>Contact selected</span>
                <button type="button" className="action-secondary" onClick={clearPickedContact} disabled={isSending}>
                  Clear
                </button>
              </div>
            ) : null}

            {showContactPicker ? (
              <div className="contact-picker-list">
                {filteredContacts.length === 0 ? (
                  <div className="contact-picker-empty">No matching contacts.</div>
                ) : (
                  filteredContacts.map((contact) => (
                    <button
                      key={contact.id}
                      type="button"
                      className={`contact-picker-item ${
                        String(contact.id) === String(selectedRecipientContactId) ? 'active' : ''
                      }`}
                      onClick={() => onContactPick(String(contact.id))}
                      disabled={isSending}
                    >
                      <strong>{contact.full_name || 'Unnamed contact'}</strong>
                      <span>{contact.phone || '-'}</span>
                      {contact.org ? <span>{contact.org}</span> : null}
                    </button>
                  ))
                )}
              </div>
            ) : null}
          </div>

          <label className="form-field">
            <span>Message</span>
            <textarea
              rows={6}
              value={message}
              onChange={(event) => setMessage(event.target.value)}
              disabled={isSending}
              placeholder="Write SMS text here"
            />
          </label>
          {defaultIdentityText ? (
            <p className="sms-limit">
              Prefix applied automatically: <strong>{defaultIdentityText}:</strong>
            </p>
          ) : null}

          <p className={`sms-limit ${smsInfo.single ? '' : 'sms-limit-error'}`}>
            {smsInfo.encoding} {smsInfo.used}/{smsInfo.max} (single SMS required)
          </p>

          <button
            type="button"
            className="template-toggle"
            onClick={() => setShowTemplateManager((prev) => !prev)}
          >
            {showTemplateManager ? 'Hide template manager' : 'Manage templates'}
          </button>

          {showTemplateManager ? (
            <>
              <h3>Template Manager</h3>
              <p className="template-hint">Create, edit or delete templates.</p>

              <div className="settings-grid">
                <label className="form-field">
                  <span>Template to edit</span>
                  <select
                    value={selectedTemplateId}
                    onChange={(event) => onTemplatePick(event.target.value)}
                    disabled={isSavingTemplate}
                  >
                    <option value="">Create new template</option>
                    {templates.map((template) => (
                      <option key={template.id} value={template.id}>
                        {template.name}
                      </option>
                    ))}
                  </select>
                </label>
              </div>

              <label className="form-field">
                <span>Template name</span>
                <input
                  type="text"
                  value={templateName}
                  onChange={(event) => setTemplateName(event.target.value)}
                  disabled={isSavingTemplate}
                  placeholder="e.g. Call back reminder"
                />
              </label>

              <label className="form-field">
                <span>Template body</span>
                <textarea
                  rows={5}
                  value={templateBody}
                  onChange={(event) => setTemplateBody(event.target.value)}
                  disabled={isSavingTemplate}
                  placeholder="Template text"
                />
              </label>

              <div className="settings-actions">
                <button type="button" className="action-primary" onClick={onSaveTemplate} disabled={isSavingTemplate}>
                  {isSavingTemplate ? 'Saving...' : selectedTemplateId ? 'Update template' : 'Save template'}
                </button>
                <button
                  type="button"
                  className="action-secondary"
                  onClick={onDeleteTemplate}
                  disabled={isSavingTemplate || !selectedTemplateId}
                >
                  Delete template
                </button>
                <button
                  type="button"
                  className="action-secondary"
                  onClick={clearTemplateForm}
                  disabled={isSavingTemplate}
                >
                  New
                </button>
                {templateMessage ? <span className="save-message">{templateMessage}</span> : null}
                {templateErrorMessage ? <span className="save-message template-error">{templateErrorMessage}</span> : null}
              </div>
            </>
          ) : null}
        </section>

        <div className="settings-actions">
          <button type="submit" className="action-primary" disabled={isSending || !message.trim()}>
            {isSending ? 'Sending...' : 'Send SMS'}
          </button>
          {statusMessage ? <span className="save-message">{statusMessage}</span> : null}
          {errorMessage ? <span className="save-message">{errorMessage}</span> : null}
        </div>
      </form>
    </section>
  )
}

export default SmsPage
