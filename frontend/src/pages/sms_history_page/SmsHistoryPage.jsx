import { useCallback, useEffect, useMemo, useState } from 'react'
import { getErrorMessage } from '../../errorUtils'
import { t } from '../../i18n'
import MessageTimeFilter from '../../components/MessageTimeFilter'
import { listSMSHistory } from '../sms_page/smsApi'
import { listContacts } from '../address_book_page/addressBookApi'
import { MESSAGE_TIME_FILTERS } from '../inbox_page/useInboxState'

const FILTER_TO_DAYS = { today: 1, week: 7, month: 30, all: 3650 }

function normalizeDate(dateValue) {
  if (!dateValue) return new Date().toISOString()
  if (dateValue.includes('T')) return dateValue
  return dateValue.replace(' ', 'T')
}

function formatDate(dateValue) {
  const date = new Date(normalizeDate(dateValue))
  return date.toLocaleString([], { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function normalizePhoneLoose(raw) {
  const source = String(raw ?? '').trim()
  if (!source) return ''
  let value = source.replace(/\D+/g, '')
  if (value.startsWith('00')) value = value.slice(2)
  if (value.startsWith('420') && value.length === 12) value = value.slice(3)
  return value
}

function SmsHistoryPage({ language = 'en' }) {
  const [timeFilter, setTimeFilter] = useState('month')
  const [items, setItems] = useState([])
  const [contacts, setContacts] = useState([])
  const [loading, setLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  const loadHistory = useCallback(async () => {
    setLoading(true)
    setErrorMessage('')
    try {
      const days = FILTER_TO_DAYS[timeFilter] ?? 30
      const payload = await listSMSHistory({ days })
      setItems(payload.items ?? [])
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'Failed to load SMS history'))
    } finally {
      setLoading(false)
    }
  }, [timeFilter])

  useEffect(() => {
    loadHistory()
  }, [loadHistory])

  useEffect(() => {
    listContacts()
      .then((payload) => setContacts(Array.isArray(payload) ? payload : []))
      .catch(() => {})
  }, [])

  const contactByPhone = useMemo(() => {
    const map = new Map()
    for (const contact of contacts) {
      const normalized = normalizePhoneLoose(contact.phone)
      if (!normalized) continue
      if (!map.has(normalized)) {
        map.set(normalized, contact)
      }
    }
    return map
  }, [contacts])

  return (
    <section>
      <header className="section-header">
        <h2>{t(language, 'sms_history_title')}</h2>
        <p>{t(language, 'sms_history_subtitle')}</p>
        <MessageTimeFilter
          value={timeFilter}
          onChange={setTimeFilter}
          filters={MESSAGE_TIME_FILTERS}
          language={language}
        />
      </header>

      <div className="sms-history-controls">
        <button type="button" className="action-secondary" onClick={loadHistory} disabled={loading}>
          {t(language, 'sms_history_refresh')}
        </button>
      </div>

      {loading ? <p className="sms-history-meta">{t(language, 'sms_history_loading')}</p> : null}
      {errorMessage ? <p className="inbox-error">{errorMessage}</p> : null}

      {!loading && !errorMessage && items.length === 0 ? (
        <div className="empty-state">{t(language, 'sms_history_empty')}</div>
      ) : null}

      {!loading && !errorMessage && items.length > 0 ? (
        <div className="sms-history-list">
          {items.map((item) => {
            const isSent = item.direction === 'sent'
            const isFailedSent = isSent && !item.success
            const number = String(item.counterparty ?? '').trim()
            const normalized = normalizePhoneLoose(number)
            const matchedContact = normalized ? contactByPhone.get(normalized) : null
            const contactName = String(matchedContact?.full_name ?? '').trim()
            return (
              <article
                key={`${item.direction}-${item.id}-${item.occurred_at || ''}-${item.counterparty || ''}`}
                className="sms-history-card"
              >
                <div className="sms-history-meta">
                  <strong>{isSent ? t(language, 'sms_history_direction_sent') : t(language, 'sms_history_direction_received')}</strong>
                  <span>{formatDate(item.occurred_at)}</span>
                </div>
                {isSent ? (
                  <p className={`sms-history-status ${isFailedSent ? 'failed' : 'sent'}`}>
                    {t(language, 'sms_history_status_label')}:{' '}
                    {isFailedSent ? t(language, 'sms_history_status_failed') : t(language, 'sms_history_status_sent')}
                  </p>
                ) : null}
                <div className="sms-history-counterparty">
                  <p className="sms-history-counterparty-name">{contactName || number || '-'}</p>
                  {contactName && number ? <p className="sms-history-counterparty-phone">{number}</p> : null}
                </div>
                <p className="sms-history-message">{item.message_text || item.subject || '-'}</p>
                {isFailedSent && item.provider_response ? (
                  <p className="inbox-error">{t(language, 'sms_history_provider_error', { value: item.provider_response })}</p>
                ) : null}
                {isFailedSent && item.error_message ? <p className="inbox-error">{item.error_message}</p> : null}
              </article>
            )
          })}
        </div>
      ) : null}
    </section>
  )
}

export default SmsHistoryPage
