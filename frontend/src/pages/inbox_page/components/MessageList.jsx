import MessageCard from './MessageCard'

function MessageList({
  messages,
  expandedIds = [],
  resolvingIds = [],
  checked = false,
  onResolve,
  onUncheck,
  onToggleExpand,
  onOpenContact,
  onSendSMS,
  showAudio = true,
}) {
  const isExpanded = (id) => expandedIds.includes(id)
  const isResolving = (id) => resolvingIds.includes(id)

  return (
    <div className="message-list">
      {messages.map((message) => (
        <MessageCard
          key={message.id}
          message={message}
          expanded={isExpanded(message.id)}
          resolving={isResolving(message.id)}
          checked={checked}
          onResolve={onResolve}
          onUncheck={onUncheck}
          onToggleExpand={onToggleExpand}
          onOpenContact={onOpenContact}
          onSendSMS={onSendSMS}
          showAudio={showAudio}
        />
      ))}
    </div>
  )
}

export default MessageList
