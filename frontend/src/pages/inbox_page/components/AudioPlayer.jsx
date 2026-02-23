import { useEffect, useState } from 'react'
import { fetchVoicemailAudioDataURL } from '../inboxApi'

const audioCache = new Map()

function AudioPlayer({ voicemailId }) {
  const [src, setSrc] = useState(() => (audioCache.has(voicemailId) ? audioCache.get(voicemailId) : ''))
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    if (!voicemailId || audioCache.has(voicemailId)) {
      if (voicemailId) setSrc(audioCache.get(voicemailId) || '')
      return
    }

    let cancelled = false
    setIsLoading(true)
    fetchVoicemailAudioDataURL(voicemailId)
      .then((url) => {
        if (cancelled) return
        audioCache.set(voicemailId, url)
        setSrc(url)
      })
      .catch(() => {
        if (cancelled) return
        setSrc('')
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [voicemailId])

  if (!voicemailId || isLoading || !src) return null

  return (
    <div className="audio-player" aria-label="Voicemail audio player">
      <audio controls preload="none" src={src}>
        Your browser does not support audio playback.
      </audio>
    </div>
  )
}

export default AudioPlayer
