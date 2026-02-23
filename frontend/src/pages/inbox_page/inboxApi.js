import * as Backend from '../../../wailsjs/go/main/App'
import { toError } from '../../errorUtils'

function normalizeDate(dateValue) {
  if (!dateValue) return new Date().toISOString()
  if (dateValue.includes('T')) return dateValue
  return dateValue.replace(' ', 'T')
}

function formatRelativeTimestamp(isoDate) {
  const date = new Date(isoDate)
  const now = new Date()
  const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const startMessageDay = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const dayDiff = Math.round((startToday.getTime() - startMessageDay.getTime()) / 86400000)
  const time = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })

  if (dayDiff === 0) return `Today, ${time}`
  if (dayDiff === 1) return `Yesterday, ${time}`
  return date.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function formatDuration(seconds) {
  if (typeof seconds !== 'number' || Number.isNaN(seconds) || seconds < 0) return '--:--s'
  const totalSeconds = Math.floor(seconds)
  const minutes = Math.floor(totalSeconds / 60)
  const remainingSeconds = totalSeconds % 60
  return `${minutes}:${String(remainingSeconds).padStart(2, '0')}s`
}

function mapVoicemailItem(item) {
  const occurredAt = normalizeDate(item.date_received)
  const contact = item.contact ?? null
  const callerLabel = contact?.full_name || item.caller_phone || 'Unknown caller'
  return {
    id: item.id,
    occurredAt,
    caller: callerLabel,
    callerPhone: item.caller_phone || '',
    contact,
    time: formatRelativeTimestamp(occurredAt),
    duration: formatDuration(item.audio_duration_s),
    transcription: item.message_text || '(No transcription available)',
    checked: Boolean(item.is_checked),
  }
}

function mapSMSItem(item) {
  const occurredAt = normalizeDate(item.date_received)
  const contact = item.contact ?? null
  const senderLabel = contact?.full_name || item.sender_phone || 'Unknown sender'
  return {
    id: item.id,
    occurredAt,
    caller: senderLabel,
    callerPhone: item.sender_phone || '',
    contact,
    time: formatRelativeTimestamp(occurredAt),
    duration: '',
    transcription: item.message_text || '(No OCR text extracted)',
    checked: Boolean(item.is_checked),
  }
}

export async function fetchVoicemails({ days, checked, version = 'all' }) {
  try {
    const payload = await Backend.ListVoicemails({
      days,
      clean: true,
      checked,
      version,
    })
    return (payload.items ?? []).map(mapVoicemailItem)
  } catch (error) {
    throw toError(error, 'Failed to load voicemails')
  }
}

export async function setVoicemailChecked(id, checked) {
  try {
    await Backend.SetVoicemailChecked(id, checked)
  } catch (error) {
    throw toError(error, 'Failed to update voicemail state')
  }
}

export async function fetchSMSMessages({ days, checked }) {
  try {
    const payload = await Backend.ListSMSMessages({
      days,
      checked,
    })
    return (payload.items ?? []).map(mapSMSItem)
  } catch (error) {
    throw toError(error, 'Failed to load SMS inbox')
  }
}

export async function setSMSMessageChecked(id, checked) {
  try {
    await Backend.SetSMSMessageChecked(id, checked)
  } catch (error) {
    throw toError(error, 'Failed to update SMS state')
  }
}

export async function triggerSync(days = 7) {
  try {
    return await Backend.SyncVoicemails(days)
  } catch (error) {
    throw toError(error, 'Failed to sync voicemails')
  }
}

export async function fetchVoicemailAudioDataURL(id) {
  try {
    return await Backend.GetVoicemailAudioDataURL(id)
  } catch (error) {
    throw toError(error, 'Failed to load audio')
  }
}
