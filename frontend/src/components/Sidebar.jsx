import { useEffect, useState } from 'react'
import { t } from '../i18n'

function formatMinutesSince(dateValue, nowValue) {
  const diffMs = Math.max(0, nowValue - dateValue)
  return Math.floor(diffMs / (60 * 1000))
}

function Sidebar({
  activeTab,
  onTabChange,
  lastCheckAt,
  connected = true,
  onResync,
  isSyncing = false,
  balanceLabel = '',
  isBalanceLoading = false,
  balanceError = '',
  language = 'en',
}) {
  const [nowTs, setNowTs] = useState(() => Date.now())

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNowTs(Date.now())
    }, 60 * 1000)
    return () => window.clearInterval(intervalId)
  }, [])

  const safeLastCheckAt = lastCheckAt ?? nowTs
  const lastCheckMinutes = formatMinutesSince(safeLastCheckAt, nowTs)

  return (
    <aside className="sidebar">
      <h1 className="brand">OdoBox</h1>
      <nav className="nav-list" aria-label="Main navigation">
        <button
          type="button"
          className={`nav-button ${activeTab === 'inbox' ? 'active' : ''}`}
          onClick={() => onTabChange('inbox')}
        >
          📨 {t(language, 'sidebar_inbox')}
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'sms' ? 'active' : ''}`}
          onClick={() => onTabChange('sms')}
        >
          ✉ {t(language, 'sidebar_sms')}
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'address-book' ? 'active' : ''}`}
          onClick={() => onTabChange('address-book')}
        >
          📖 {t(language, 'sidebar_address_book')}
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'help' ? 'active' : ''}`}
          onClick={() => onTabChange('help')}
        >
          ❓ {t(language, 'sidebar_help')}
        </button>
      </nav>

      <section className="sidebar-bottom">
        <button
          type="button"
          className={`nav-button secondary-nav-button ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => onTabChange('settings')}
        >
          ⚙ {t(language, 'sidebar_settings')}
        </button>

        <div className="sidebar-status">
          <p className="sync-title">⟳ {t(language, 'sidebar_sync')}</p>
          <p className="sync-meta">{t(language, 'sidebar_last_check', { minutes: lastCheckMinutes })}</p>
          <p className="sync-meta">
            {t(language, 'sidebar_credit', {
              value:
                isBalanceLoading
                  ? t(language, 'sidebar_credit_loading')
                  : balanceLabel || (balanceError ? t(language, 'sidebar_credit_unavailable') : '--'),
            })}
          </p>
          <p className="sync-meta">● {connected ? t(language, 'sidebar_connected') : t(language, 'sidebar_error')}</p>
          <button type="button" className="sync-link" onClick={onResync} disabled={isSyncing}>
            {isSyncing ? t(language, 'sidebar_resyncing') : t(language, 'sidebar_resync')}
          </button>
        </div>
      </section>
    </aside>
  )
}

export default Sidebar
