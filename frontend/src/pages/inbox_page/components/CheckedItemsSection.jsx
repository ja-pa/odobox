import MessageList from './MessageList'

function CheckedItemsSection({ items, onUncheck, onOpenContact, onSendSMS, showAudio = true }) {
  if (items.length === 0) return null

  return (
    <section className="checked-section">
      <h3>Checked items</h3>
      <MessageList
        messages={items}
        checked
        onUncheck={onUncheck}
        onOpenContact={onOpenContact}
        onSendSMS={onSendSMS}
        showAudio={showAudio}
      />
    </section>
  )
}

export default CheckedItemsSection
