import { useMemo, useState } from 'react'
import { normalizeLanguage, t } from '../../i18n'

const HELP_IMAGES = {
  lineSettings: '/help/linka_nastaveni.png',
  lineDetails: '/help/nastaveni_konkretni_linky.png',
  sounds: '/help/zvukove_hlasky.png',
  voicemail: '/help/hlasova_schranka.png',
  api: '/help/nastaveni_api.png',
}

const HELP_COPY = {
  cs: {
    title: 'Nápověda',
    subtitle: 'Nastavení Odoriku pro OdoBox (hlasové zprávy, SMS přes e-mail, API přístup).',
    languageLabel: 'Jazyk',
    tocTitle: 'Obsah',
    sections: [
      {
        title: '1) Nastavení e-mailu pro příjem dat',
        body: 'OdoBox čte zprávy, které Odorik posílá e-mailem. U linky musí být správně vyplněný e-mail, který OdoBox sleduje přes IMAP.',
        items: [
          'V administraci Odorik otevřete sekci Linky.',
          'Otevřete detail konkrétní linky (např. 781245).',
          'Do pole „Email pro SMS, faxy a hlasové zprávy“ zadejte např. odobox.demo.mailbox@gmail.com.',
          'Stejnou adresu nastavte i do „Email pro zasílání nahraných hovorů“.',
        ],
        images: [HELP_IMAGES.lineSettings, HELP_IMAGES.lineDetails],
      },
      {
        title: '2) Správa a výběr uvítací hlášky',
        body: 'Aby volající slyšel pozdrav před zanecháním vzkazu, je potřeba mít hlášku nahranou a přiřazenou k lince.',
        items: [
          'V Nastavení účtu otevřete „Správa zvukových hlášek“ a zkontrolujte číslo hlášky (např. 1).',
          'Hlášku můžete nahrát z PC nebo vytočením kódu *0091 z linky.',
          'V detailu linky vyberte v položce „Přehrání uvítací hlášky“ správnou hlášku.',
        ],
        images: [HELP_IMAGES.sounds, HELP_IMAGES.lineDetails],
      },
      {
        title: '3) Aktivace hlasové schránky s přepisem',
        body: 'Odorik může posílat MP3 záznam i textový přepis hlasového vzkazu.',
        items: [
          'V Průvodci nastavením otevřete „Paralelní vyzvánění“.',
          'Přidejte kód: *085100*1',
          '*0851 = hlasová schránka se záznamníkem',
          '00 = okamžité přesměrování (0 vteřin)',
          '*1 = přehrání hlášky číslo 1',
        ],
        images: [HELP_IMAGES.voicemail],
      },
      {
        title: '4) Příjem SMS zpráv',
        body: 'U nomadických a geografických čísel Odorik převádí příchozí SMS do PDF. Pokud je e-mail správně nastavený u linky, další konfigurace není nutná.',
      },
      {
        title: '5) API přístup pro OdoBox',
        body: 'Pro odesílání SMS a kontrolu kreditu je nutné vyplnit API přístupy Odorik v OdoBox Nastavení.',
        items: [
          'V Odorik otevřete Nastavení účtu -> API heslo.',
          'Najdete tam API uživatele (např. 7012345) a API heslo (např. apidemo123).',
          'Tyto údaje vyplňte v OdoBox Nastavení -> Odorik API Access.',
        ],
        images: [HELP_IMAGES.api],
      },
      {
        title: '6) Gmail IMAP přístup (heslo aplikace)',
        body: 'Pokud používáte Gmail schránku, běžné heslo účtu nestačí. Je nutné zapnout dvoufázové ověření a vytvořit Heslo pro aplikaci.',
        items: [
          'V Google účtu otevřete Zabezpečení a zapněte dvoufázové ověření.',
          'Poté otevřete stránku Hesla pro aplikace (App passwords).',
          'Vytvořte nové heslo aplikace pro „Mail“ (název např. OdoBox).',
          'V OdoBox Nastavení -> IMAP vyplňte: host imap.gmail.com, port 993, SSL zapnuto.',
          'Do IMAP username zadejte celou Gmail adresu a do IMAP password vložte vygenerované heslo aplikace.',
        ],
      },
    ],
    summaryTitle: 'Souhrn nastavení',
    summaryRows: [
      ['Kód paralelního vyzvánění', '*085100*1'],
      ['E-mail pro zprávy', 'odobox.demo.mailbox@gmail.com'],
      ['Uvítací hláška', 'Vybrat v detailu linky'],
      ['API ID', '7012345'],
      ['API heslo', 'apidemo123'],
    ],
  },
  en: {
    title: 'Help',
    subtitle:
      'How to configure Odorik for OdoBox (voicemail, SMS via email, and API credentials).',
    languageLabel: 'Language',
    tocTitle: 'Contents',
    sections: [
      {
        title: '1) Email setup for incoming data',
        body: 'OdoBox reads messages sent by Odorik to email. Each line must have a mailbox configured, and that mailbox must be the one OdoBox checks through IMAP.',
        items: [
          'Open the Lines section in Odorik administration.',
          'Open details of a specific line (for example 781245).',
          'Set “Email for SMS, faxes and voicemail” to a mailbox such as odobox.demo.mailbox@gmail.com.',
          'Set the same address for “Email for recorded calls”.',
        ],
        images: [HELP_IMAGES.lineSettings, HELP_IMAGES.lineDetails],
      },
      {
        title: '2) Greeting recording management',
        body: 'To play a greeting before voicemail recording, upload a greeting and assign it to the line.',
        items: [
          'In Account Settings, open “Sound recordings management” and check greeting number (for example 1).',
          'Upload from PC or record via code *0091 from your line.',
          'In line details, select the correct greeting in “Play greeting message”.',
        ],
        images: [HELP_IMAGES.sounds, HELP_IMAGES.lineDetails],
      },
      {
        title: '3) Enable voicemail with transcript',
        body: 'Odorik can send both MP3 audio and text transcription of voicemail.',
        items: [
          'Open “Parallel ringing” in the setup wizard.',
          'Add code: *085100*1',
          '*0851 = voicemail/answering service',
          '00 = immediate forwarding (0 seconds)',
          '*1 = play greeting number 1',
        ],
        images: [HELP_IMAGES.voicemail],
      },
      {
        title: '4) Incoming SMS',
        body: 'For nomadic/geographic numbers, Odorik converts incoming SMS to PDF automatically. If line email is configured correctly, no extra setup is required.',
      },
      {
        title: '5) API access for OdoBox',
        body: 'API credentials are required for sending SMS and checking balance.',
        items: [
          'In Odorik open Account Settings -> API password.',
          'You will find API user (for example 7012345) and API password (for example apidemo123).',
          'Fill these values in OdoBox Settings -> Odorik API Access.',
        ],
        images: [HELP_IMAGES.api],
      },
      {
        title: '6) Gmail IMAP access (App Password)',
        body: 'If you use a Gmail mailbox, your regular account password is not enough. You need 2-Step Verification enabled and an App Password generated for OdoBox.',
        items: [
          'Open Google Account -> Security and enable 2-Step Verification.',
          'Then open App passwords.',
          'Create a new app password for “Mail” (name it e.g. OdoBox).',
          'In OdoBox Settings -> IMAP use: host imap.gmail.com, port 993, SSL enabled.',
          'Use full Gmail address as IMAP username and paste the generated app password into IMAP password.',
        ],
      },
    ],
    summaryTitle: 'Configuration summary',
    summaryRows: [
      ['Parallel ringing code', '*085100*1'],
      ['Message mailbox', 'odobox.demo.mailbox@gmail.com'],
      ['Greeting message', 'Select in line details'],
      ['API user ID', '7012345'],
      ['API password', 'apidemo123'],
    ],
  },
}

function HelpPage({ language = 'en' }) {
  const lang = normalizeLanguage(language)
  const [previewImage, setPreviewImage] = useState(null)
  const copy = useMemo(() => HELP_COPY[lang] ?? HELP_COPY.en, [lang])

  return (
    <section>
      <header className="section-header">
        <h2>{t(lang, 'help_title')}</h2>
        <p>{t(lang, 'help_subtitle')}</p>
      </header>

      <section className="settings-card">
        <h3>{copy.tocTitle}</h3>
        <ul className="help-list">
          {copy.sections.map((section, index) => (
            <li key={`toc-${section.title}`}>
              <a href={`#help-section-${index + 1}`}>{section.title}</a>
            </li>
          ))}
        </ul>
      </section>

      <div className="settings-layout help-layout">
        {copy.sections.map((section, index) => (
          <section className="settings-card" id={`help-section-${index + 1}`} key={section.title}>
            <h3>{section.title}</h3>
            <p className="help-note">{section.body}</p>
            {section.items?.length ? (
              <ul className="help-list">
                {section.items.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            ) : null}
            {section.images?.length ? (
              <div className="help-image-grid">
                {section.images.map((src) => (
                  <button
                    type="button"
                    key={src}
                    className="help-image-link"
                    onClick={() => setPreviewImage({ src, title: section.title })}
                  >
                    <img src={src} alt={section.title} className="help-image" loading="lazy" />
                  </button>
                ))}
              </div>
            ) : null}
          </section>
        ))}

        <section className="settings-card">
          <h3>{copy.summaryTitle}</h3>
          <div className="help-summary-table">
            <div className="help-summary-row help-summary-head">
              <strong>{t(lang, 'help_summary_parameter')}</strong>
              <strong>{t(lang, 'help_summary_value')}</strong>
            </div>
            {copy.summaryRows.map(([name, value]) => (
              <div className="help-summary-row" key={`${name}-${value}`}>
                <span>{name}</span>
                <span>{value}</span>
              </div>
            ))}
          </div>
        </section>
      </div>

      {previewImage ? (
        <div className="help-lightbox" role="dialog" aria-modal="true" onClick={() => setPreviewImage(null)}>
          <div className="help-lightbox-card" onClick={(event) => event.stopPropagation()}>
            <button type="button" className="help-lightbox-close" onClick={() => setPreviewImage(null)}>
              {lang === 'cs' ? 'Zavřít' : 'Close'}
            </button>
            <img src={previewImage.src} alt={previewImage.title} className="help-lightbox-image" />
          </div>
        </div>
      ) : null}
    </section>
  )
}

export default HelpPage
