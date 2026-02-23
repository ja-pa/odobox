export const SETTINGS_STORAGE_KEY = 'odorik_central_settings'

export const DEFAULT_SETTINGS = {
  imap: {
    host: '',
    port: '993',
    username: '',
    password: '',
    mailbox: 'INBOX',
    secure: true,
  },
  odorikPin: '',
  pollIntervalMinutes: '5',
  transcriptVersion: 'v2',
}

export function loadStoredSettings() {
  const storedValue = window.localStorage.getItem(SETTINGS_STORAGE_KEY)
  if (!storedValue) return DEFAULT_SETTINGS

  try {
    const parsed = JSON.parse(storedValue)
    return {
      imap: {
        ...DEFAULT_SETTINGS.imap,
        ...(parsed.imap || {}),
      },
      odorikPin: parsed.odorikPin ?? DEFAULT_SETTINGS.odorikPin,
      pollIntervalMinutes: parsed.pollIntervalMinutes ?? DEFAULT_SETTINGS.pollIntervalMinutes,
      transcriptVersion: parsed.transcriptVersion ?? DEFAULT_SETTINGS.transcriptVersion,
    }
  } catch {
    return DEFAULT_SETTINGS
  }
}

export function saveSettings(settings) {
  window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings))
}

export function clearSettings() {
  window.localStorage.removeItem(SETTINGS_STORAGE_KEY)
}
