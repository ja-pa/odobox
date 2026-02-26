import { t } from '../i18n'

function MessageTimeFilter({ value, onChange, filters = [], language = 'en' }) {
  return (
    <div className="time-filter-list" role="tablist" aria-label={t(language, 'time_filter_aria')}>
      {filters.map((filterKey) => (
        <button
          key={filterKey}
          type="button"
          role="tab"
          aria-selected={value === filterKey}
          className={`time-filter-pill ${value === filterKey ? 'active' : ''}`}
          onClick={() => onChange(filterKey)}
        >
          {t(language, `time_filter_${filterKey}`)}
        </button>
      ))}
    </div>
  )
}

export default MessageTimeFilter
