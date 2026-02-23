import * as Backend from '../wailsjs/go/main/App'
import { toError } from './errorUtils'

const DEFAULT_KEEP_LINE_REGEX = String.raw`^v\d+:\s*.+$`
const DEFAULT_V1_REGEX = String.raw`(?is)(?:^|\n)\s*v1:\s*(?P<content>.*?)(?=\n\s*v2:\s*|\Z)`
const DEFAULT_V2_REGEX = String.raw`(?is)(?:^|\n)\s*v2:\s*(?P<content>.*?)(?=\n\s*v1:\s*|\Z)`
const DEFAULT_CALLER_REGEX = String.raw`^Hlasova zprava\s+(\+?\d+)\s+[-=]+>\s+\d+`
const DEFAULT_SMS_TEXT_REGEX = String.raw`(?is)TEXT:\s*["“]?(?:Message)?(?P<content>[^"\r\n]+?)["”]`

export const DEFAULT_SETTINGS = {
  imap: {
    host: '',
    port: '993',
    username: '',
    password: '',
    mailbox: 'INBOX',
    secure: true,
  },
  odorik: {
    pin: '',
    user: '',
    password: '',
    senderId: '',
  },
  pollIntervalMinutes: '5',
  transcriptVersion: 'v2',
  smsIdentityText: '',
  messageCleaner: {
    keepLineRegex: DEFAULT_KEEP_LINE_REGEX,
    versionV1Regex: DEFAULT_V1_REGEX,
    versionV2Regex: DEFAULT_V2_REGEX,
    removeRegexes: '',
    collapseBlankLines: true,
  },
  voicemailParser: {
    callerPhoneRegex: DEFAULT_CALLER_REGEX,
  },
  smsParser: {
    textExtractRegex: DEFAULT_SMS_TEXT_REGEX,
  },
}

function parseBoolean(value, fallback = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value !== 'string') return fallback
  return ['1', 'true', 'yes', 'on'].includes(value.trim().toLowerCase())
}

function normalizeRegexListValue(value) {
  if (Array.isArray(value)) {
    return value
      .map((item) => String(item).trim())
      .filter(Boolean)
      .join('\n')
  }
  if (typeof value !== 'string') return ''
  return value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .join('\n')
}

function normalizeTranscriptVersion(value) {
  const normalized = String(value ?? '').trim().toLowerCase()
  return ['v1', 'v2', 'both'].includes(normalized) ? normalized : DEFAULT_SETTINGS.transcriptVersion
}

function toRegexList(value) {
  return String(value ?? '')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

function mapApiToSettings(apiSettings = {}) {
  const imap = apiSettings.imap ?? {}
  const cleaner = apiSettings.message_cleaner ?? {}
  const parser = apiSettings.voicemail_parser ?? {}
  const smsParser = apiSettings.sms_parser ?? {}
  const app = apiSettings.app ?? {}
  const odorik = apiSettings.odorik ?? {}

  return {
    imap: {
      host: imap.host ?? DEFAULT_SETTINGS.imap.host,
      port: String(imap.port ?? DEFAULT_SETTINGS.imap.port),
      username: imap.username ?? DEFAULT_SETTINGS.imap.username,
      password: imap.password ?? DEFAULT_SETTINGS.imap.password,
      mailbox: imap.folder ?? DEFAULT_SETTINGS.imap.mailbox,
      secure: parseBoolean(imap.ssl, DEFAULT_SETTINGS.imap.secure),
    },
    odorik: {
      pin: odorik.pin ?? DEFAULT_SETTINGS.odorik.pin,
      user:
        odorik.user ?? odorik.account_id ?? DEFAULT_SETTINGS.odorik.user,
      password:
        odorik.password ?? odorik.api_pin ?? odorik.pin ?? DEFAULT_SETTINGS.odorik.password,
      senderId: odorik.sender_id ?? DEFAULT_SETTINGS.odorik.senderId,
    },
    pollIntervalMinutes: String(app.poll_interval_minutes ?? DEFAULT_SETTINGS.pollIntervalMinutes),
    transcriptVersion: normalizeTranscriptVersion(
      app.default_transcript_version ?? DEFAULT_SETTINGS.transcriptVersion
    ),
    smsIdentityText: app.sms_identity_text ?? DEFAULT_SETTINGS.smsIdentityText,
    messageCleaner: {
      keepLineRegex: cleaner.keep_line_regex ?? DEFAULT_SETTINGS.messageCleaner.keepLineRegex,
      versionV1Regex: cleaner.version_v1_regex ?? DEFAULT_SETTINGS.messageCleaner.versionV1Regex,
      versionV2Regex: cleaner.version_v2_regex ?? DEFAULT_SETTINGS.messageCleaner.versionV2Regex,
      removeRegexes: normalizeRegexListValue(cleaner.remove_regexes),
      collapseBlankLines: parseBoolean(
        cleaner.collapse_blank_lines,
        DEFAULT_SETTINGS.messageCleaner.collapseBlankLines
      ),
    },
    voicemailParser: {
      callerPhoneRegex:
        parser.caller_phone_regex ?? DEFAULT_SETTINGS.voicemailParser.callerPhoneRegex,
    },
    smsParser: {
      textExtractRegex:
        smsParser.text_extract_regex ?? DEFAULT_SETTINGS.smsParser.textExtractRegex,
    },
  }
}

function mapSettingsToApi(settings, editableSections = []) {
  const sections = new Set(editableSections)
  const payload = {}

  if (sections.has('imap')) {
    payload.imap = {
      host: settings.imap.host,
      port: Number.parseInt(settings.imap.port, 10) || 993,
      username: settings.imap.username,
      password: settings.imap.password,
      folder: settings.imap.mailbox,
      ssl: Boolean(settings.imap.secure),
    }
  }

  if (sections.has('app')) {
    payload.app = {
      poll_interval_minutes: Number.parseInt(settings.pollIntervalMinutes, 10) || 5,
      default_transcript_version: settings.transcriptVersion,
      sms_identity_text: settings.smsIdentityText ?? '',
    }
  }

  if (sections.has('odorik')) {
    payload.odorik = {
      pin: settings.odorik.password || settings.odorik.pin,
      user: settings.odorik.user,
      password: settings.odorik.password,
      account_id: settings.odorik.user,
      api_pin: settings.odorik.password,
      sender_id: settings.odorik.senderId,
    }
  }

  if (sections.has('message_cleaner')) {
    payload.message_cleaner = {
      keep_line_regex: settings.messageCleaner.keepLineRegex,
      version_v1_regex: settings.messageCleaner.versionV1Regex,
      version_v2_regex: settings.messageCleaner.versionV2Regex,
      remove_regexes: toRegexList(settings.messageCleaner.removeRegexes),
      collapse_blank_lines: Boolean(settings.messageCleaner.collapseBlankLines),
    }
  }

  if (sections.has('voicemail_parser')) {
    payload.voicemail_parser = {
      caller_phone_regex: settings.voicemailParser.callerPhoneRegex,
    }
  }

  if (sections.has('sms_parser')) {
    payload.sms_parser = {
      text_extract_regex: settings.smsParser.textExtractRegex,
    }
  }

  return payload
}

export async function fetchSettingsFromApi() {
  try {
    const payload = await Backend.GetSettings()
    return {
      settings: mapApiToSettings(payload.settings ?? {}),
      editableSections: Array.isArray(payload.editable_sections) ? payload.editable_sections : [],
    }
  } catch (error) {
    throw toError(error, 'Failed to load settings')
  }
}

export async function saveSettingsToApi(settings, editableSections = []) {
  const settingsPayload = mapSettingsToApi(settings, editableSections)
  try {
    const payload = await Backend.PatchSettings({ settings: settingsPayload })
    return mapApiToSettings(payload.settings ?? {})
  } catch (error) {
    throw toError(error, 'Failed to save settings')
  }
}
