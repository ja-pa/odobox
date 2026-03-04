import { useEffect, useState } from 'react'
import { createSMSTemplate, deleteSMSTemplate, listSMSTemplates, updateSMSTemplate } from '../sms_page/smsApi'
import { t } from '../../i18n'

function SmsTemplatePage({ language = 'en' }) {
  const [templates, setTemplates] = useState([])
  const [selectedTemplateId, setSelectedTemplateId] = useState('')
  const [templateName, setTemplateName] = useState('')
  const [templateBody, setTemplateBody] = useState('')
  const [isSavingTemplate, setIsSavingTemplate] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  const loadTemplates = async () => {
    const items = await listSMSTemplates()
    setTemplates(Array.isArray(items) ? items : [])
  }

  useEffect(() => {
    loadTemplates().catch((error) => setErrorMessage(error.message))
  }, [])

  const clearTemplateForm = () => {
    setSelectedTemplateId('')
    setTemplateName('')
    setTemplateBody('')
  }

  const onTemplatePick = (value) => {
    setSelectedTemplateId(value)
    setStatusMessage('')
    setErrorMessage('')
    if (!value) {
      setTemplateName('')
      setTemplateBody('')
      return
    }
    const found = templates.find((item) => String(item.id) === String(value))
    if (!found) return
    setTemplateName(found.name || '')
    setTemplateBody(found.body || '')
  }

  const onSaveTemplate = async () => {
    setStatusMessage('')
    setErrorMessage('')
    if (!templateName.trim()) {
      setErrorMessage(t(language, 'sms_template_error_name_required'))
      return
    }
    if (!templateBody.trim()) {
      setErrorMessage(t(language, 'sms_template_error_body_required'))
      return
    }

    setIsSavingTemplate(true)
    try {
      if (selectedTemplateId) {
        const updated = await updateSMSTemplate({
          id: Number(selectedTemplateId),
          name: templateName,
          body: templateBody,
        })
        setStatusMessage(t(language, 'sms_template_status_updated', { name: updated.name }))
      } else {
        const created = await createSMSTemplate({ name: templateName, body: templateBody })
        setStatusMessage(t(language, 'sms_template_status_created', { name: created.name }))
        setSelectedTemplateId(String(created.id))
      }
      await loadTemplates()
    } catch (error) {
      setErrorMessage(error.message)
    } finally {
      setIsSavingTemplate(false)
    }
  }

  const onDeleteTemplate = async () => {
    if (!selectedTemplateId) {
      setErrorMessage(t(language, 'sms_template_error_select_delete'))
      return
    }
    setIsSavingTemplate(true)
    setStatusMessage('')
    setErrorMessage('')
    try {
      await deleteSMSTemplate(Number(selectedTemplateId))
      await loadTemplates()
      clearTemplateForm()
      setStatusMessage(t(language, 'sms_template_status_deleted'))
    } catch (error) {
      setErrorMessage(error.message)
    } finally {
      setIsSavingTemplate(false)
    }
  }

  return (
    <section>
      <header className="section-header">
        <h2>{t(language, 'sms_template_title')}</h2>
        <p>{t(language, 'sms_template_subtitle')}</p>
      </header>

      <section className="settings-card">
        <div className="settings-grid">
          <label className="form-field">
            <span>{t(language, 'sms_template_edit_label')}</span>
            <select
              value={selectedTemplateId}
              onChange={(event) => onTemplatePick(event.target.value)}
              disabled={isSavingTemplate}
            >
              <option value="">{t(language, 'sms_template_new_option')}</option>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
          </label>
        </div>

        <label className="form-field">
          <span>{t(language, 'sms_template_name_label')}</span>
          <input
            type="text"
            value={templateName}
            onChange={(event) => setTemplateName(event.target.value)}
            disabled={isSavingTemplate}
            placeholder={t(language, 'sms_template_name_placeholder')}
          />
        </label>

        <label className="form-field">
          <span>{t(language, 'sms_template_body_label')}</span>
          <textarea
            rows={5}
            value={templateBody}
            onChange={(event) => setTemplateBody(event.target.value)}
            disabled={isSavingTemplate}
            placeholder={t(language, 'sms_template_body_placeholder')}
          />
        </label>

        <div className="settings-actions">
          <button type="button" className="action-primary" onClick={onSaveTemplate} disabled={isSavingTemplate}>
            {isSavingTemplate
              ? t(language, 'common_saving')
              : selectedTemplateId
              ? t(language, 'sms_template_update')
              : t(language, 'sms_template_save')}
          </button>
          <button
            type="button"
            className="action-secondary"
            onClick={onDeleteTemplate}
            disabled={isSavingTemplate || !selectedTemplateId}
          >
            {t(language, 'sms_template_delete')}
          </button>
          <button type="button" className="action-secondary" onClick={clearTemplateForm} disabled={isSavingTemplate}>
            {t(language, 'sms_template_new')}
          </button>
          {statusMessage ? <span className="save-message">{statusMessage}</span> : null}
          {errorMessage ? <span className="save-message template-error">{errorMessage}</span> : null}
        </div>
      </section>
    </section>
  )
}

export default SmsTemplatePage
