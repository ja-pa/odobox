import { useEffect, useState } from 'react'

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
          📨 Inbox
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'sms' ? 'active' : ''}`}
          onClick={() => onTabChange('sms')}
        >
          ✉ SMS
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'address-book' ? 'active' : ''}`}
          onClick={() => onTabChange('address-book')}
        >
          📖 Address Book
        </button>
        <button
          type="button"
          className={`nav-button ${activeTab === 'help' ? 'active' : ''}`}
          onClick={() => onTabChange('help')}
        >
          ❓ Help
        </button>
      </nav>

      <section className="sidebar-bottom">
        <button
          type="button"
          className={`nav-button secondary-nav-button ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => onTabChange('settings')}
        >
          ⚙ Settings
        </button>

        <div className="sidebar-status">
          <p className="sync-title">⟳ Sync</p>
          <p className="sync-meta">Last check {lastCheckMinutes} min ago</p>
          <p className="sync-meta">
            Credit{' '}
            {isBalanceLoading
              ? 'loading...'
              : balanceLabel || (balanceError ? 'unavailable' : '--')}
          </p>
          <p className="sync-meta">● {connected ? 'Connected' : 'Error'}</p>
          <button type="button" className="sync-link" onClick={onResync} disabled={isSyncing}>
            {isSyncing ? 'Resyncing...' : 'Resync now'}
          </button>
        </div>
      </section>
    </aside>
  )
}

export default Sidebar
