import { useState } from 'react'
import { getErrorMessage } from '../../errorUtils'
import { updateContact } from '../address_book_page/addressBookApi'
import CheckedItemsSection from './components/CheckedItemsSection'
import ContactModal from './components/ContactModal'
import MessageList from './components/MessageList'
import { MESSAGE_TIME_FILTERS } from './useInboxState'

const FILTER_LABELS = {
  today: 'Today',
  week: 'Week',
  month: 'Month',
  all: 'All time',
}

function InboxPage({ inboxState, onSendSMSContact }) {
  const {
    resolvedMessages,
    resolvedSMSMessages,
    resolvingIds,
    resolvingSMSIds,
    expandedIds,
    expandedSMSIds,
    showCheckedItems,
    filteredVisibleMessages,
    filteredVisibleSMSMessages,
    timeFilter,
    isLoading,
    errorMessage,
    reloadMessages,
    toggleExpand,
    toggleSMSExpand,
    handleResolve,
    handleUnresolve,
    handleResolveSMS,
    handleUnresolveSMS,
    setShowCheckedItems,
    setTimeFilter,
  } = inboxState
  const totalChecked = resolvedMessages.length + resolvedSMSMessages.length

  const [selectedMessage, setSelectedMessage] = useState(null)
  const [isSavingContact, setIsSavingContact] = useState(false)
  const [contactStatusMessage, setContactStatusMessage] = useState('')
  const [contactErrorMessage, setContactErrorMessage] = useState('')

  const openContact = (message) => {
    if (!message?.contact) return
    setSelectedMessage(message)
    setContactStatusMessage('')
    setContactErrorMessage('')
  }

  const closeContact = () => {
    if (isSavingContact) return
    setSelectedMessage(null)
    setContactStatusMessage('')
    setContactErrorMessage('')
  }

  const saveContact = async (form) => {
    if (!selectedMessage?.contact?.id) return
    setIsSavingContact(true)
    setContactStatusMessage('')
    setContactErrorMessage('')
    try {
      const updated = await updateContact({
        id: selectedMessage.contact.id,
        ...form,
      })
      setSelectedMessage((prev) =>
        prev
          ? {
              ...prev,
              caller: updated.full_name || prev.caller,
              callerPhone: updated.phone || prev.callerPhone,
              contact: updated,
            }
          : prev
      )
      await reloadMessages()
      setContactStatusMessage('Contact updated.')
    } catch (error) {
      setContactErrorMessage(getErrorMessage(error, 'Failed to update contact'))
    } finally {
      setIsSavingContact(false)
    }
  }

  return (
    <section>
      <header className="section-header">
        <h2>Inbox</h2>
        <p>Resolve each call entry to keep a zero inbox.</p>
        <button
          type="button"
          className="show-checked-button"
          disabled={totalChecked === 0}
          onClick={() => setShowCheckedItems((prev) => !prev)}
        >
          {showCheckedItems ? 'Hide checked items' : 'Show checked items'}
          {totalChecked > 0 ? ` (${totalChecked})` : ''}
        </button>
        <div className="time-filter-list" role="tablist" aria-label="Message time filter">
          {MESSAGE_TIME_FILTERS.map((filterKey) => (
            <button
              key={filterKey}
              type="button"
              role="tab"
              aria-selected={timeFilter === filterKey}
              className={`time-filter-pill ${timeFilter === filterKey ? 'active' : ''}`}
              onClick={() => setTimeFilter(filterKey)}
            >
              {FILTER_LABELS[filterKey]}
            </button>
          ))}
        </div>
      </header>

      {isLoading ? (
        <div className="empty-state">Loading messages...</div>
      ) : errorMessage ? (
        <div className="empty-state inbox-error">{errorMessage}</div>
      ) : filteredVisibleMessages.length === 0 && filteredVisibleSMSMessages.length === 0 ? (
        <div className="empty-state">All caught up! Your inbox is empty.</div>
      ) : (
        <>
          {filteredVisibleMessages.length > 0 ? (
            <MessageList
              messages={filteredVisibleMessages}
              expandedIds={expandedIds}
              resolvingIds={resolvingIds}
              onResolve={handleResolve}
              onToggleExpand={toggleExpand}
              onOpenContact={openContact}
              onSendSMS={onSendSMSContact}
            />
          ) : null}
          {filteredVisibleSMSMessages.length > 0 ? (
            <section className="sms-inbox-section">
              <h3>SMS Inbox</h3>
              <MessageList
                messages={filteredVisibleSMSMessages}
                expandedIds={expandedSMSIds}
                resolvingIds={resolvingSMSIds}
                onResolve={handleResolveSMS}
                onToggleExpand={toggleSMSExpand}
                onOpenContact={openContact}
                onSendSMS={onSendSMSContact}
                showAudio={false}
              />
            </section>
          ) : null}
        </>
      )}

      {showCheckedItems && resolvedMessages.length > 0 ? (
        <CheckedItemsSection
          items={resolvedMessages}
          onUncheck={handleUnresolve}
          onOpenContact={openContact}
          onSendSMS={onSendSMSContact}
        />
      ) : null}
      {showCheckedItems && resolvedSMSMessages.length > 0 ? (
        <section className="sms-inbox-section">
          <h3>Checked SMS</h3>
          <CheckedItemsSection
            items={resolvedSMSMessages}
            onUncheck={handleUnresolveSMS}
            onOpenContact={openContact}
            onSendSMS={onSendSMSContact}
            showAudio={false}
          />
        </section>
      ) : null}

      <ContactModal
        message={selectedMessage}
        isSaving={isSavingContact}
        statusMessage={contactStatusMessage}
        errorMessage={contactErrorMessage}
        onClose={closeContact}
        onSave={saveContact}
      />
    </section>
  )
}

export default InboxPage
