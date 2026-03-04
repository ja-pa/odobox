import { useEffect, useMemo, useState } from 'react'
import {
  getSMSLengthInfo,
  listSMSTemplates,
  sendSMS,
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

  const applyTemplateToMessage = (templateId) => {
    if (!templateId) return
    const found = templates.find((item) => String(item.id) === String(templateId))
    if (!found) return
    setMessage(found.body || '')
  }

  const onTemplatePick = (value) => {
    setSelectedTemplateId(value)
    if (!value) {
      setMessage('')
      return
    }
    applyTemplateToMessage(value)
  }

  const onSubmit = async (event) => {
    event.preventDefault()
    setErrorMessage('')
    setStatusMessage('')

    if (!recipient.trim()) {
      setErrorMessage(t(language, 'sms_error_recipient_required'))
      return
    }
    if (!message.trim()) {
      setErrorMessage(t(language, 'sms_error_message_required'))
      return
    }
    if (!smsInfo.single) {
      setErrorMessage(
        t(language, 'sms_error_too_long', { encoding: smsInfo.encoding, used: smsInfo.used, max: smsInfo.max })
      )
      return
    }

    setIsSending(true)
    try {
      const result = await sendSMS({ recipient, message, sender })
      setStatusMessage(
        t(language, 'sms_status_sent', {
          recipient: result.recipient,
          encoding: result.encoding,
          used: result.chars_used,
          max: result.max_single_chars,
        })
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
          <h3>{t(language, 'sms_form_title')}</h3>

          <label className="form-field">
            <span>{t(language, 'sms_template_optional')}</span>
            <select
              value={selectedTemplateId}
              onChange={(event) => onTemplatePick(event.target.value)}
              disabled={isSending}
            >
              <option value="">{t(language, 'sms_no_template')}</option>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
          </label>

          <div className="settings-grid">
            <label className="form-field">
              <span>{t(language, 'sms_recipient_label')}</span>
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
              <span>{t(language, 'sms_sender_label')}</span>
              <input
                type="text"
                placeholder={t(language, 'sms_sender_placeholder')}
                value={sender}
                onChange={(event) => setSender(event.target.value)}
                disabled={isSending}
              />
            </label>
          </div>

          <div className="contact-picker">
            <label className="form-field">
              <span>{t(language, 'sms_find_recipient')}</span>
              <input
                type="text"
                placeholder={t(language, 'sms_find_recipient_placeholder')}
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
                <span>{t(language, 'sms_contact_selected')}</span>
                <button type="button" className="action-secondary" onClick={clearPickedContact} disabled={isSending}>
                  {t(language, 'sms_clear_contact')}
                </button>
              </div>
            ) : null}

            {showContactPicker ? (
              <div className="contact-picker-list">
                {filteredContacts.length === 0 ? (
                  <div className="contact-picker-empty">{t(language, 'sms_no_matching_contacts')}</div>
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
                      <strong>{contact.full_name || t(language, 'sms_unnamed_contact')}</strong>
                      <span>{contact.phone || '-'}</span>
                      {contact.org ? <span>{contact.org}</span> : null}
                    </button>
                  ))
                )}
              </div>
            ) : null}
          </div>

          <label className="form-field">
            <span>{t(language, 'sms_message_label')}</span>
            <textarea
              rows={6}
              value={message}
              onChange={(event) => setMessage(event.target.value)}
              disabled={isSending}
              placeholder={t(language, 'sms_message_placeholder')}
            />
          </label>
          {defaultIdentityText ? (
            <p className="sms-limit">
              {t(language, 'sms_prefix_applied')} <strong>{defaultIdentityText}:</strong>
            </p>
          ) : null}

          <p className={`sms-limit ${smsInfo.single ? '' : 'sms-limit-error'}`}>
            {t(language, 'sms_limit_single', { encoding: smsInfo.encoding, used: smsInfo.used, max: smsInfo.max })}
          </p>

        </section>

        <div className="settings-actions">
          <button type="submit" className="action-primary" disabled={isSending || !message.trim()}>
            {isSending ? t(language, 'sms_sending') : t(language, 'sms_send_button')}
          </button>
          {statusMessage ? <span className="save-message">{statusMessage}</span> : null}
          {errorMessage ? <span className="save-message">{errorMessage}</span> : null}
        </div>
      </form>
    </section>
  )
}

export default SmsPage
