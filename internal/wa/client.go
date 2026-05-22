package wa

import (
	"context"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func init() {
	// Ask WhatsApp to ship the full chat history (not just recent N days)
	// on first pair. Only takes effect at pairing time.
	store.DeviceProps.RequireFullSync = proto.Bool(true)
}

// Build constructs a connected whatsmeow client. If no session exists it
// blocks until QR pairing succeeds (printing the code to stdout).
func Build(ctx context.Context, dataDir string, debug bool) (*whatsmeow.Client, error) {
	dbPath := filepath.Join(dataDir, "session.db")
	level := "INFO"
	if debug {
		level = "DEBUG"
	}
	dbLog := waLog.Stdout("DB", level, true)
	container, err := sqlstore.New(ctx, "sqlite3",
		fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
	if err != nil {
		return nil, fmt.Errorf("sqlstore: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	client := whatsmeow.NewClient(device, waLog.Stdout("WA", level, true))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("connect: %w", err)
		}
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				printQR(evt.Code)
			case "success":
				fmt.Println("Paired successfully.")
			case "timeout":
				return nil, fmt.Errorf("QR pairing timed out")
			case "err-client-outdated":
				return nil, fmt.Errorf("client outdated")
			default:
				fmt.Printf("QR event: %s\n", evt.Event)
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("connect: %w", err)
		}
	}
	return client, nil
}

func printQR(code string) {
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		fmt.Println("QR generate error:", err)
		fmt.Println("Raw code:", code)
		return
	}
	fmt.Println("\nScan this QR with WhatsApp (Settings → Linked Devices → Link a Device):")
	fmt.Println(qr.ToSmallString(false))
}
