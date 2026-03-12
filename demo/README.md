Use this folder for a safe demo setup.

Generate the demo database:

```bash
make demo-db
```

Run the app against the demo database:

```bash
ODORIK_CONFIG=demo/config.ini make dev
```

Or point only the database path at runtime:

```bash
ODORIK_DB=demo/demo.db make dev
```

The seeded database contains:

- 5 demo contacts
- 3 received SMS messages
- 3 sent SMS history items
- 2 SMS templates
- 3 demo voicemails
