import AudioPlayer from './AudioPlayer'

const EXPAND_THRESHOLD = 180

function MessageCard({
  message,
  expanded = false,
  resolving = false,
  checked = false,
  showAudio = true,
  onResolve,
  onUncheck,
  onToggleExpand,
  onOpenContact,
  onSendSMS,
}) {
  const canExpand = !checked && message.transcription.length > EXPAND_THRESHOLD
  const hasContact = Boolean(message.contact)
  const canSendSMS = hasContact && Boolean(message.callerPhone)

  return (
    <article className={`message-card ${resolving ? 'resolving' : ''} ${checked ? 'checked-card' : ''}`}>
      <label className="resolve-control" aria-label={`${checked ? 'Resolved' : 'Resolve'} ${message.caller}`}>
        <input
          type="checkbox"
          checked={checked || resolving}
          disabled={resolving}
          onChange={() => {
            if (checked) {
              onUncheck?.(message.id)
              return
            }
            onResolve?.(message.id)
          }}
        />
      </label>

      <div className="message-main">
        <div className="message-header">
          <div>
            {hasContact ? (
              <button
                type="button"
                className="contact-link"
                onClick={() => onOpenContact?.(message)}
              >
                {message.caller}
              </button>
            ) : (
              <h3>{message.caller}</h3>
            )}
            {message.callerPhone ? <p className="contact-sub">{message.callerPhone}</p> : null}
          </div>
          <div className="metadata">
            {canSendSMS ? (
              <button
                type="button"
                className="send-sms-icon"
                title="Send SMS to this contact"
                aria-label={`Send SMS to ${message.caller}`}
                onClick={() => onSendSMS?.(message.callerPhone, message.caller)}
              >
                ✉
              </button>
            ) : null}
            <span>{message.time}</span>
            {message.duration ? <span>{message.duration}</span> : null}
          </div>
        </div>
        <p className={`transcription ${canExpand && !expanded ? 'truncated' : ''}`}>
          {message.transcription}
        </p>
        {canExpand ? (
          <button type="button" className="view-more" onClick={() => onToggleExpand?.(message.id)}>
            {expanded ? 'View Less' : 'View More'}
          </button>
        ) : null}
      </div>

      {showAudio ? <AudioPlayer voicemailId={message.id} /> : null}
    </article>
  )
}

export default MessageCard
