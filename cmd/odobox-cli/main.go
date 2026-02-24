package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	imapadapter "OdorikCentral/internal/adapters/imap"
	ocradapter "OdorikCentral/internal/adapters/ocr"
	sqliteadapter "OdorikCentral/internal/adapters/sqlite"
	"OdorikCentral/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		printCLIUsage()
		os.Exit(2)
	}

	b := core.NewBackendWithDeps("", core.BackendDeps{
		MailGatewayFactory: imapadapter.NewFactory(),
		OCRService:         ocradapter.NewService(),
		SyncStoreFactory:   sqliteadapter.NewFactory(),
	})
	var err error

	switch os.Args[1] {
	case "list":
		err = runCLIList(b, os.Args[2:])
	case "list-sms":
		err = runCLIListSMS(b, os.Args[2:])
	case "fetch":
		err = runCLIFetch(b, os.Args[2:])
	case "debug-imap":
		err = runCLIDebugIMAP(b, os.Args[2:])
	case "debug-imap-message":
		err = runCLIDebugIMAPMessage(b, os.Args[2:])
	case "paths":
		err = runCLIPaths(b, os.Args[2:])
	case "help", "-h", "--help":
		printCLIUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printCLIUsage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printCLIUsage() {
	fmt.Println("odobox-cli - OdoBox command line interface")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  odobox-cli <command> [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list   List saved voicemail messages from SQLite")
	fmt.Println("  list-sms List saved inbound SMS messages from SQLite")
	fmt.Println("  fetch  Fetch new voicemail messages from IMAP")
	fmt.Println("  debug-imap Inspect recent IMAP envelope data for sender/subject matching")
	fmt.Println("  debug-imap-message Inspect MIME parts for one IMAP UID")
	fmt.Println("  paths  Show resolved config/db paths")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  odobox-cli list --days 14")
	fmt.Println("  odobox-cli list-sms --days 14")
	fmt.Println("  odobox-cli list --checked true --version v2")
	fmt.Println("  odobox-cli fetch --days 3")
	fmt.Println("  odobox-cli debug-imap --days 7 --limit 60")
	fmt.Println("  odobox-cli debug-imap-message --seq 1")
	fmt.Println("  odobox-cli paths")
}

func runCLIList(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	days := fs.Int("days", 7, "How many past days to include")
	clean := fs.Bool("clean", true, "Apply transcript cleaner/version extraction")
	checked := fs.String("checked", "all", "Filter by checked state: all|true|false")
	version := fs.String("version", "", "Transcript version: all|v1|v2 (empty uses config default)")
	asJSON := fs.Bool("json", false, "Output full response as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resp, err := b.ListVoicemails(core.ListVoicemailsRequest{
		Days:    *days,
		Clean:   *clean,
		Checked: *checked,
		Version: *version,
	})
	if err != nil {
		return err
	}

	if *asJSON {
		return printJSON(resp)
	}

	fmt.Printf("Found %d messages (clean=%t, version=%s)\n", resp.Count, resp.Clean, resp.Version)
	fmt.Println("ID  DATE                 CALLER       CONTACT                CHK  DUR  SUBJECT / MESSAGE")
	for _, item := range resp.Items {
		caller := "-"
		if item.CallerPhone != nil && strings.TrimSpace(*item.CallerPhone) != "" {
			caller = *item.CallerPhone
		}
		contact := "-"
		if item.Contact != nil && strings.TrimSpace(item.Contact.FullName) != "" {
			contact = item.Contact.FullName
		}
		chk := "no"
		if item.IsChecked {
			chk = "yes"
		}
		dur := "-"
		if item.AudioSeconds != nil {
			dur = fmt.Sprintf("%ds", *item.AudioSeconds)
		}
		summary := strings.TrimSpace(item.Subject)
		msgSnippet := clipWhitespace(item.MessageText, 52)
		if msgSnippet != "" {
			if summary != "" {
				summary += " | "
			}
			summary += msgSnippet
		}
		fmt.Printf("%-3d %-19s %-12s %-22s %-4s %-4s %s\n",
			item.ID,
			clipWhitespace(item.DateReceived, 19),
			clipWhitespace(caller, 12),
			clipWhitespace(contact, 22),
			chk,
			dur,
			summary,
		)
	}

	return nil
}

func runCLIFetch(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	days := fs.Int("days", 7, "Fetch messages newer than this many days")
	asJSON := fs.Bool("json", false, "Output full response as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resp, err := b.Sync(*days)
	if err != nil {
		return err
	}

	if *asJSON {
		return printJSON(resp)
	}

	fmt.Printf("Fetch completed: status=%s days=%d stored=%d skipped_duplicates=%d\n",
		resp.Status, resp.Days, resp.Stored, resp.SkippedDuplicates)
	fmt.Printf("  Voicemail: stored=%d skipped=%d\n", resp.VoicemailStored, resp.VoicemailSkipped)
	fmt.Printf("  SMS:       stored=%d skipped=%d\n", resp.SMSStored, resp.SMSSkipped)
	return nil
}

func runCLIListSMS(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("list-sms", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	days := fs.Int("days", 7, "How many past days to include")
	checked := fs.String("checked", "all", "Filter by checked state: all|true|false")
	asJSON := fs.Bool("json", false, "Output full response as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resp, err := b.ListSMSMessages(core.ListSMSMessagesRequest{
		Days:    *days,
		Checked: *checked,
	})
	if err != nil {
		return err
	}

	if *asJSON {
		return printJSON(resp)
	}

	fmt.Printf("Found %d SMS inbox messages\n", resp.Count)
	fmt.Println("ID  DATE                 SENDER       CONTACT                CHK  SUBJECT / OCR")
	for _, item := range resp.Items {
		sender := "-"
		if item.SenderPhone != nil && strings.TrimSpace(*item.SenderPhone) != "" {
			sender = *item.SenderPhone
		}
		contact := "-"
		if item.Contact != nil && strings.TrimSpace(item.Contact.FullName) != "" {
			contact = item.Contact.FullName
		}
		chk := "no"
		if item.IsChecked {
			chk = "yes"
		}
		summary := strings.TrimSpace(item.Subject)
		msgSnippet := clipWhitespace(item.MessageText, 52)
		if msgSnippet != "" {
			if summary != "" {
				summary += " | "
			}
			summary += msgSnippet
		}
		fmt.Printf("%-3d %-19s %-12s %-22s %-4s %s\n",
			item.ID,
			clipWhitespace(item.DateReceived, 19),
			clipWhitespace(sender, 12),
			clipWhitespace(contact, 22),
			chk,
			summary,
		)
	}
	return nil
}

func runCLIDebugIMAP(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("debug-imap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	days := fs.Int("days", 7, "How many past days to inspect")
	limit := fs.Int("limit", 60, "Maximum recent messages to inspect")
	asJSON := fs.Bool("json", false, "Output full response as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	items, err := b.DebugIMAP(*days, *limit)
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(items)
	}

	fmt.Printf("IMAP debug: inspected=%d days=%d\n", len(items), *days)
	fmt.Println("SEQ      UID      DATE                 SMS  VM   ODORIK  FROM                               SUBJECT")
	for _, item := range items {
		sms := "no"
		if item.SMSLike {
			sms = "yes"
		}
		vm := "no"
		if item.Voicemail {
			vm = "yes"
		}
		host := "no"
		if item.HasFromHost {
			host = "yes"
		}
		fmt.Printf("%-8d %-8d %-19s %-4s %-4s %-6s %-34s %s\n",
			item.Seq,
			item.UID,
			clipWhitespace(item.Date, 19),
			sms,
			vm,
			host,
			clipWhitespace(item.From, 34),
			clipWhitespace(item.Subject, 64),
		)
	}
	return nil
}

func runCLIDebugIMAPMessage(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("debug-imap-message", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	seq := fs.Uint("seq", 0, "Message sequence number to inspect (see debug-imap output)")
	asJSON := fs.Bool("json", false, "Output full response as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	msg, err := b.DebugIMAPMessage(uint32(*seq))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(msg)
	}
	fmt.Printf("SEQ: %d\n", msg.Seq)
	fmt.Printf("UID: %d\n", msg.UID)
	fmt.Printf("Date: %s\n", msg.Date)
	fmt.Printf("From: %s\n", msg.From)
	fmt.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("Parts: %d\n", len(msg.Parts))
	for i, p := range msg.Parts {
		fmt.Printf("  %d) kind=%s type=%s file=%s size=%d\n", i+1, p.Kind, p.ContentType, p.Filename, p.SizeBytes)
		if p.Sample != "" {
			fmt.Printf("     sample=%s\n", p.Sample)
		}
	}
	return nil
}

func runCLIPaths(b *core.Backend, args []string) error {
	fs := flag.NewFlagSet("paths", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "Output resolved paths as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfgPath := b.ResolveConfigPath()
	dbPath, err := b.ResolveDBPathFromCurrentConfig()
	if err != nil {
		return err
	}

	payload := map[string]any{
		"config_path": cfgPath,
		"db_path":     dbPath,
	}
	if *asJSON {
		return printJSON(payload)
	}

	fmt.Printf("Config path: %s\n", cfgPath)
	fmt.Printf("DB path:     %s\n", dbPath)
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func clipWhitespace(raw string, maxLen int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if maxLen <= 0 {
		return ""
	}
	r := []rune(trimmed)
	if len(r) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return "."
	}
	return string(r[:maxLen-3]) + "..."
}
