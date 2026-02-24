import MessageList from './MessageList'
import { t } from '../../../i18n'

function CheckedItemsSection({ items, onUncheck, onOpenContact, onSendSMS, showAudio = true, language = 'en' }) {
  if (items.length === 0) return null

  return (
    <section className="checked-section">
      <h3>{t(language, 'checked_items_title')}</h3>
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
