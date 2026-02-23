import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  fetchSMSMessages,
  fetchVoicemails,
  setSMSMessageChecked,
  setVoicemailChecked,
  triggerSync,
} from './inboxApi'
import { getErrorMessage } from '../../errorUtils'

export const MESSAGE_TIME_FILTERS = ['today', 'week', 'month', 'all']
const FILTER_TO_DAYS = { today: 1, week: 7, month: 30, all: 3650 }

function sortMessagesByDateTime(items) {
  return [...items].sort((a, b) => new Date(b.occurredAt).getTime() - new Date(a.occurredAt).getTime())
}

function startOfDay(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function filterMessagesByTime(messages, timeFilter) {
  if (timeFilter === 'all') return messages

  const now = new Date()
  const todayStart = startOfDay(now)
  const tomorrowStart = new Date(todayStart)
  tomorrowStart.setDate(tomorrowStart.getDate() + 1)

  if (timeFilter === 'today') {
    return messages.filter((message) => {
      const messageDate = new Date(message.occurredAt)
      if (Number.isNaN(messageDate.getTime())) return false
      return messageDate >= todayStart && messageDate < tomorrowStart
    })
  }

  const daysBackByFilter = { week: 6, month: 29 }
  const daysBack = daysBackByFilter[timeFilter]
  if (typeof daysBack !== 'number') return messages

  const rangeStart = new Date(todayStart)
  rangeStart.setDate(rangeStart.getDate() - daysBack)

  return messages.filter((message) => {
    const messageDate = new Date(message.occurredAt)
    if (Number.isNaN(messageDate.getTime())) return false
    return messageDate >= rangeStart && messageDate < tomorrowStart
  })
}

function toPollIntervalMs(value) {
  const parsed = Number.parseInt(value, 10)
  if (!Number.isFinite(parsed) || parsed < 1) return 5 * 60 * 1000
  return parsed * 60 * 1000
}

function useInboxState({ pollIntervalMinutes = '5', transcriptVersion = 'v2' } = {}) {
  const [messages, setMessages] = useState([])
  const [resolvedMessages, setResolvedMessages] = useState([])
  const [smsMessages, setSMSMessages] = useState([])
  const [resolvedSMSMessages, setResolvedSMSMessages] = useState([])
  const [resolvingIds, setResolvingIds] = useState([])
  const [resolvingSMSIds, setResolvingSMSIds] = useState([])
  const [expandedIds, setExpandedIds] = useState([])
  const [expandedSMSIds, setExpandedSMSIds] = useState([])
  const [showCheckedItems, setShowCheckedItems] = useState(false)
  const [timeFilter, setTimeFilter] = useState('today')
  const [isLoading, setIsLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [lastSyncAt, setLastSyncAt] = useState(() => Date.now())
  const [syncErrorMessage, setSyncErrorMessage] = useState('')
  const [isSyncing, setIsSyncing] = useState(false)
  const syncInProgressRef = useRef(false)
  const didInitialSyncRef = useRef(false)
  const pollIntervalMs = useMemo(() => toPollIntervalMs(pollIntervalMinutes), [pollIntervalMinutes])

  const loadMessages = useCallback(
    async ({ signal, withLoading = true } = {}) => {
      if (withLoading) setIsLoading(true)
      setErrorMessage('')
      try {
        const days = FILTER_TO_DAYS[timeFilter]
        const [unchecked, checked, uncheckedSMS, checkedSMS] = await Promise.all([
          fetchVoicemails({ days, checked: 'false', version: transcriptVersion, signal }),
          fetchVoicemails({ days, checked: 'true', version: transcriptVersion, signal }),
          fetchSMSMessages({ days, checked: 'false', signal }),
          fetchSMSMessages({ days, checked: 'true', signal }),
        ])
        setMessages(sortMessagesByDateTime(filterMessagesByTime(unchecked, timeFilter)))
        setResolvedMessages(sortMessagesByDateTime(filterMessagesByTime(checked, timeFilter)))
        setSMSMessages(sortMessagesByDateTime(filterMessagesByTime(uncheckedSMS, timeFilter)))
        setResolvedSMSMessages(sortMessagesByDateTime(filterMessagesByTime(checkedSMS, timeFilter)))
      } catch (error) {
        if (error.name === 'AbortError') return
        setErrorMessage(getErrorMessage(error, 'Failed to load messages'))
      } finally {
        if (!signal?.aborted && withLoading) setIsLoading(false)
      }
    },
    [timeFilter, transcriptVersion]
  )

  useEffect(() => {
    const controller = new AbortController()
    loadMessages({ signal: controller.signal, withLoading: true })
    return () => controller.abort()
  }, [loadMessages])

  const runSync = useCallback(async () => {
    if (syncInProgressRef.current) return
    syncInProgressRef.current = true
    setIsSyncing(true)
    try {
      const days = FILTER_TO_DAYS[timeFilter]
      await triggerSync(days)
      setLastSyncAt(Date.now())
      setSyncErrorMessage('')
      await loadMessages({ withLoading: false })
    } catch (error) {
      setSyncErrorMessage(getErrorMessage(error, 'Failed to sync'))
    } finally {
      syncInProgressRef.current = false
      setIsSyncing(false)
    }
  }, [loadMessages, timeFilter])

  useEffect(() => {
    if (didInitialSyncRef.current) return
    didInitialSyncRef.current = true
    runSync()
  }, [runSync])

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      runSync()
    }, pollIntervalMs)
    return () => window.clearInterval(intervalId)
  }, [pollIntervalMs, runSync])

  const toggleExpand = (id) => {
    setExpandedIds((prev) =>
      prev.includes(id) ? prev.filter((itemId) => itemId !== id) : [...prev, id]
    )
  }

  const toggleSMSExpand = (id) => {
    setExpandedSMSIds((prev) =>
      prev.includes(id) ? prev.filter((itemId) => itemId !== id) : [...prev, id]
    )
  }

  const handleResolve = async (id) => {
    if (resolvingIds.includes(id)) return
    const resolvedItem = messages.find((message) => message.id === id)
    if (!resolvedItem) return

    setResolvingIds((prev) => [...prev, id])
    window.setTimeout(async () => {
      try {
        await setVoicemailChecked(id, true)
      } catch (error) {
        setErrorMessage(getErrorMessage(error, 'Failed to update voicemail'))
        setResolvingIds((prev) => prev.filter((itemId) => itemId !== id))
        return
      }
      setResolvedMessages((prev) =>
        prev.some((message) => message.id === id) ? prev : [resolvedItem, ...prev]
      )
      setMessages((prev) => prev.filter((message) => message.id !== id))
      setResolvingIds((prev) => prev.filter((itemId) => itemId !== id))
      setExpandedIds((prev) => prev.filter((itemId) => itemId !== id))
    }, 280)
  }

  const handleUnresolve = async (id) => {
    const restoredItem = resolvedMessages.find((message) => message.id === id)
    if (!restoredItem) return

    try {
      await setVoicemailChecked(id, false)
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'Failed to update voicemail'))
      return
    }
    setResolvedMessages((prev) => prev.filter((message) => message.id !== id))
    setMessages((prev) =>
      prev.some((message) => message.id === id) ? prev : sortMessagesByDateTime([...prev, restoredItem])
    )
  }

  const handleResolveSMS = async (id) => {
    if (resolvingSMSIds.includes(id)) return
    const resolvedItem = smsMessages.find((message) => message.id === id)
    if (!resolvedItem) return

    setResolvingSMSIds((prev) => [...prev, id])
    window.setTimeout(async () => {
      try {
        await setSMSMessageChecked(id, true)
      } catch (error) {
        setErrorMessage(getErrorMessage(error, 'Failed to update SMS'))
        setResolvingSMSIds((prev) => prev.filter((itemId) => itemId !== id))
        return
      }
      setResolvedSMSMessages((prev) =>
        prev.some((message) => message.id === id) ? prev : [resolvedItem, ...prev]
      )
      setSMSMessages((prev) => prev.filter((message) => message.id !== id))
      setResolvingSMSIds((prev) => prev.filter((itemId) => itemId !== id))
      setExpandedSMSIds((prev) => prev.filter((itemId) => itemId !== id))
    }, 280)
  }

  const handleUnresolveSMS = async (id) => {
    const restoredItem = resolvedSMSMessages.find((message) => message.id === id)
    if (!restoredItem) return

    try {
      await setSMSMessageChecked(id, false)
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'Failed to update SMS'))
      return
    }
    setResolvedSMSMessages((prev) => prev.filter((message) => message.id !== id))
    setSMSMessages((prev) =>
      prev.some((message) => message.id === id) ? prev : sortMessagesByDateTime([...prev, restoredItem])
    )
  }

  useEffect(() => {
    if (resolvedMessages.length === 0 && resolvedSMSMessages.length === 0) {
      setShowCheckedItems(false)
    }
  }, [resolvedMessages, resolvedSMSMessages])

  const reloadMessages = useCallback(async () => {
    await loadMessages({ withLoading: false })
  }, [loadMessages])

  const visibleMessages = useMemo(
    () => messages.filter((message) => !resolvingIds.includes(message.id)),
    [messages, resolvingIds]
  )

  const visibleSMSMessages = useMemo(
    () => smsMessages.filter((message) => !resolvingSMSIds.includes(message.id)),
    [smsMessages, resolvingSMSIds]
  )

  return {
    messages,
    resolvedMessages,
    smsMessages,
    resolvedSMSMessages,
    resolvingIds,
    resolvingSMSIds,
    expandedIds,
    expandedSMSIds,
    showCheckedItems,
    filteredVisibleMessages: visibleMessages,
    filteredVisibleSMSMessages: visibleSMSMessages,
    timeFilter,
    isLoading,
    errorMessage,
    lastSyncAt,
    isSyncing,
    syncErrorMessage,
    runSync,
    reloadMessages,
    toggleExpand,
    toggleSMSExpand,
    handleResolve,
    handleUnresolve,
    handleResolveSMS,
    handleUnresolveSMS,
    setShowCheckedItems,
    setTimeFilter,
  }
}

export default useInboxState
