CREATE TABLE voicemails (
    id INTEGER PRIMARY KEY,
    message_id TEXT UNIQUE,
    date_received DATETIME,
    subject TEXT,
    message_text TEXT,
    is_checked INTEGER NOT NULL DEFAULT 0,
    attachment_name TEXT,
    attachment_data BLOB,
    audio_duration INTEGER
);

CREATE TABLE sms_outbox (
    id INTEGER PRIMARY KEY,
    created_at DATETIME NOT NULL,
    recipient TEXT NOT NULL,
    sender_id TEXT,
    message_text TEXT NOT NULL,
    encoding TEXT NOT NULL,
    chars_used INTEGER NOT NULL,
    max_chars INTEGER NOT NULL,
    provider_response TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE TABLE sms_templates (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE contacts (
    id INTEGER PRIMARY KEY,
    full_name TEXT NOT NULL,
    phone_display TEXT NOT NULL,
    phone_normalized TEXT NOT NULL UNIQUE,
    email TEXT,
    org TEXT,
    note TEXT,
    vcard TEXT,
    source TEXT NOT NULL DEFAULT 'manual',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE sms_inbox (
    id INTEGER PRIMARY KEY,
    message_id TEXT UNIQUE,
    date_received DATETIME,
    subject TEXT,
    sender_phone TEXT,
    message_text TEXT,
    attachment_text TEXT,
    is_checked INTEGER NOT NULL DEFAULT 0,
    attachment_name TEXT,
    attachment_data BLOB
);

INSERT INTO contacts (id, full_name, phone_display, phone_normalized, email, org, note, vcard, source, created_at, updated_at) VALUES
    (1, 'Alice Demo', '+420 777 100 101', '420777100101', 'alice@example.test', 'Demo Bakery', 'Main demo contact for inbound SMS and voicemail.', '', 'manual', '2026-03-01 08:10:00', '2026-03-01 08:10:00'),
    (2, 'Bob Support', '+420 608 200 300', '420608200300', 'bob@example.test', 'Support Desk', 'Used in sent SMS history.', '', 'manual', '2026-03-01 08:12:00', '2026-03-01 08:12:00'),
    (3, 'Charlie Courier', '731 555 222', '731555222', 'charlie@example.test', 'Courier Co', 'Demonstrates local Czech number without country code.', '', 'manual', '2026-03-01 08:15:00', '2026-03-01 08:15:00'),
    (4, 'Delta Office', '+44 20 7946 0958', '442079460958', 'delta@example.test', 'London Branch', 'Shows that non-CZ contacts also work.', '', 'manual', '2026-03-01 08:20:00', '2026-03-01 08:20:00'),
    (5, 'Neutral Caller', '+420 777 123 456', '420777123456', 'neutral@example.test', 'Sample Calls', 'Sanitized voicemail example adapted from the real database.', '', 'manual', '2026-03-01 08:25:00', '2026-03-01 08:25:00');

INSERT INTO sms_templates (id, name, body, created_at, updated_at) VALUES
    (1, 'Call back reminder', 'Dobrý den, ozveme se Vám dnes odpoledne. Děkujeme.', '2026-03-01 09:00:00', '2026-03-01 09:00:00'),
    (2, 'Order ready', 'Vaše objednávka je připravena k vyzvednutí. Otevírací doba je 8-17.', '2026-03-01 09:05:00', '2026-03-01 09:05:00');

INSERT INTO sms_inbox (id, message_id, date_received, subject, sender_phone, message_text, attachment_text, is_checked, attachment_name, attachment_data) VALUES
    (1, 'demo-sms-in-1', '2026-03-09 08:15:00', 'SMS od 777100101', '777100101', 'TEXT: Dobrý den, můžete mi zavolat po 14. hodině?', 'TEXT: Dobrý den, můžete mi zavolat po 14. hodině?', 0, '', NULL),
    (2, 'demo-sms-in-2', '2026-03-08 16:42:00', 'SMS od 731555222', '731555222', 'TEXT: Kurýr dorazí za 20 minut.', 'TEXT: Kurýr dorazí za 20 minut.', 1, '', NULL),
    (3, 'demo-sms-in-3', '2026-03-07 11:05:00', 'SMS od 447700900123', '447700900123', 'TEXT: Please confirm tomorrow''s meeting.', 'TEXT: Please confirm tomorrow''s meeting.', 0, '', NULL);

INSERT INTO sms_outbox (id, created_at, recipient, sender_id, message_text, encoding, chars_used, max_chars, provider_response, success, error_message) VALUES
    (1, '2026-03-09 09:00:00', '420608200300', 'demo', 'Dobrý den, zavoláme vám dnes mezi 15:00 a 16:00.', 'UCS-2', 49, 70, 'successfully_sent 123.45', 1, ''),
    (2, '2026-03-08 14:25:00', '420777100101', '', 'Objednávka je připravena k vyzvednutí.', 'UCS-2', 37, 70, 'successfully_sent 124.45', 1, ''),
    (3, '2026-03-07 07:55:00', '447700900123', 'demo', 'Test message for provider error example.', 'GSM-7', 39, 160, 'error forbidden_sender', 0, 'error forbidden_sender');

INSERT INTO voicemails (id, message_id, date_received, subject, message_text, is_checked, attachment_name, attachment_data, audio_duration) VALUES
    (1, 'demo-vm-1', '2026-03-08 10:10:58', 'Hlasova zprava 777123456 -> 910000111 delka 4s', 'Prehrana zprava cislo 1-Hlasova schranka, voicemail v priloze.

--- Přepis hlasové zprávy (google_v2) ---
Jo, jenom jsem chtěl říct, že v pondělí nedojdu.

Více informací o přepisu nahrávky na text:
https://forum.odorik.cz/viewtopic.php?p=46775#p46775', 0, 'vm-neutral.mp3', NULL, 4),
    (2, 'demo-vm-2', '2026-03-08 18:03:00', 'Hlasova zprava 608200300 -> 910124813 delka 7s', 'Prehrana zprava cislo 1-Hlasova schranka, voicemail v priloze.

--- Přepis hlasové zprávy (google_v2) ---
Potřebuji potvrdit termín servisu.

v1: Potrebuji potvrdit termin servisu.', 1, 'vm-bob.mp3', NULL, 7),
    (3, 'demo-vm-3', '2026-03-06 12:45:00', 'Hlasova zprava 731555222 -> 910124813 delka 5s', 'Prehrana zprava cislo 1-Hlasova schranka, voicemail v priloze.

--- Přepis hlasové zprávy (google_v2) ---
Kurýr je před budovou.

v1: Kuryr je pred budovou.', 0, 'vm-charlie.mp3', NULL, 5);
