import { useCallback, useEffect, useState } from 'react'
import Sidebar from './components/Sidebar'
import AddressBookPage from './pages/address_book_page/AddressBookPage'
import HelpPage from './pages/help_page/HelpPage'
import InboxPage from './pages/inbox_page/InboxPage'
import SmsPage from './pages/sms_page/SmsPage'
import SmsHistoryPage from './pages/sms_history_page/SmsHistoryPage'
import SmsTemplatePage from './pages/sms_template_page/SmsTemplatePage'
import useInboxState from './pages/inbox_page/useInboxState'
import SettingPage from './pages/setting_page/SettingPage'
import { DEFAULT_SETTINGS, fetchSettingsFromApi, saveSettingsToApi } from './settingsApi'
import { getErrorMessage } from './errorUtils'
import { fetchOdorikBalance } from './balanceApi'
import { normalizeLanguage } from './i18n'
import './App.css'

function App() {
  const [activeTab, setActiveTab] = useState('inbox')
  const [settings, setSettings] = useState(DEFAULT_SETTINGS)
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsErrorMessage, setSettingsErrorMessage] = useState('')
  const [editableSections, setEditableSections] = useState([])
  const [smsPrefill, setSmsPrefill] = useState({ recipient: '', label: '', token: 0 })
  const [balanceLabel, setBalanceLabel] = useState('')
  const [isBalanceLoading, setIsBalanceLoading] = useState(false)
  const [balanceError, setBalanceError] = useState('')
  const uiLanguage = normalizeLanguage(settings.uiLanguage)

  const inboxState = useInboxState({
    pollIntervalMinutes: settings.pollIntervalMinutes,
    transcriptVersion: settings.transcriptVersion,
  })

  useEffect(() => {
    const loadSettings = async () => {
      setSettingsLoading(true)
      setSettingsErrorMessage('')
      try {
        const payload = await fetchSettingsFromApi()
        setSettings(payload.settings)
        setEditableSections(payload.editableSections)
      } catch (error) {
        setSettingsErrorMessage(getErrorMessage(error, 'Failed to load settings'))
      } finally {
        setSettingsLoading(false)
      }
    }
    loadSettings()
  }, [])

  const canLoadBalance = Boolean(settings?.odorik?.user && settings?.odorik?.password)

  const refreshBalance = useCallback(
    async ({ withLoading = true } = {}) => {
      if (!canLoadBalance) {
        setBalanceLabel('')
        setBalanceError('')
        setIsBalanceLoading(false)
        return
      }
      if (withLoading) setIsBalanceLoading(true)
      try {
        const payload = await fetchOdorikBalance()
        const amount = String(payload.balance ?? '').trim()
        const currency = String(payload.currency ?? '').trim()
        setBalanceLabel([amount, currency].filter(Boolean).join(' ').trim())
        setBalanceError('')
      } catch (error) {
        setBalanceError(getErrorMessage(error, 'Failed to load Odorik balance'))
      } finally {
        if (withLoading) setIsBalanceLoading(false)
      }
    },
    [canLoadBalance]
  )

  useEffect(() => {
    if (!canLoadBalance) return
    refreshBalance({ withLoading: true })
    const intervalId = window.setInterval(() => {
      refreshBalance({ withLoading: false })
    }, 5 * 60 * 1000)
    return () => window.clearInterval(intervalId)
  }, [canLoadBalance, refreshBalance])

  const openSMSForContact = (recipient, label = '') => {
    if (!recipient) return
    setSmsPrefill({
      recipient,
      label,
      token: Date.now(),
    })
    setActiveTab('sms-compose')
  }

  const handleResync = async () => {
    await inboxState.runSync()
    refreshBalance({ withLoading: false })
  }

  const renderPage = () => {
    if (activeTab === 'inbox') {
      return <InboxPage inboxState={inboxState} onSendSMSContact={openSMSForContact} language={uiLanguage} />
    }
    if (activeTab === 'sms' || activeTab === 'sms-compose') {
      return (
        <SmsPage
          defaultSenderId={settings.odorik.senderId}
          defaultIdentityText={settings.smsIdentityText}
          prefillRecipient={smsPrefill.recipient}
          prefillLabel={smsPrefill.label}
          prefillToken={smsPrefill.token}
          language={uiLanguage}
        />
      )
    }
    if (activeTab === 'sms-history') {
      return <SmsHistoryPage language={uiLanguage} />
    }
    if (activeTab === 'sms-template') {
      return <SmsTemplatePage language={uiLanguage} />
    }
    if (activeTab === 'settings') {
      return (
        <SettingPage
          initialSettings={settings}
          isLoading={settingsLoading}
          errorMessage={settingsErrorMessage}
          onSaveSettings={async (nextSettings) => {
            setSettingsErrorMessage('')
            try {
              const savedSettings = await saveSettingsToApi(nextSettings, editableSections)
              setSettings(savedSettings)
            } catch (error) {
              setSettingsErrorMessage(getErrorMessage(error, 'Failed to save settings'))
              throw error
            }
          }}
          onReloadSettings={async () => {
            setSettingsLoading(true)
            setSettingsErrorMessage('')
            try {
              const payload = await fetchSettingsFromApi()
              setSettings(payload.settings)
              setEditableSections(payload.editableSections)
            } catch (error) {
              setSettingsErrorMessage(getErrorMessage(error, 'Failed to reload settings'))
              throw error
            } finally {
              setSettingsLoading(false)
            }
          }}
          language={uiLanguage}
        />
      )
    }
    if (activeTab === 'address-book') return <AddressBookPage onSendSMSContact={openSMSForContact} language={uiLanguage} />
    if (activeTab === 'help') return <HelpPage language={uiLanguage} />
    return <InboxPage inboxState={inboxState} onSendSMSContact={openSMSForContact} language={uiLanguage} />
  }

  return (
    <div className="app-shell">
      <Sidebar
        activeTab={activeTab}
        onTabChange={setActiveTab}
        lastCheckAt={inboxState.lastSyncAt}
        connected={!inboxState.syncErrorMessage}
        onResync={handleResync}
        isSyncing={inboxState.isSyncing}
        balanceLabel={balanceLabel}
        isBalanceLoading={isBalanceLoading}
        balanceError={balanceError}
        language={uiLanguage}
      />

      <main className="main-content">{renderPage()}</main>
    </div>
  )
}

export default App
