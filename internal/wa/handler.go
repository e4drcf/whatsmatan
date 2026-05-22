package wa

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/e4drcf/whatsmatan/internal/chats"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Tracker reports whether transcripts should be produced for a given chat JID.
type Tracker interface {
	Has(jid string) bool
}

// Transcriber turns OGG/Opus audio bytes into text.
type Transcriber interface {
	Run(ctx context.Context, ogg []byte) (string, error)
}

// ChatCache records observed chats so the web UI can list them.
type ChatCache interface {
	Upsert(chats.Entry) error
}

type Handler struct {
	Client      *whatsmeow.Client
	Tracker     Tracker
	Transcriber Transcriber
	Chats       ChatCache
	Timeout     time.Duration // per-message transcribe budget
}

// Handle is the entry point passed to client.AddEventHandler.
func (h *Handler) Handle(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.onMessage(v)
	case *events.HistorySync:
		h.onHistorySync(v)
	}
}

func (h *Handler) onMessage(msg *events.Message) {
	h.recordChat(msg)

	log.Printf("msg: chat=%s sender=%s isFromMe=%v hasAudio=%v",
		msg.Info.Chat, msg.Info.Sender, msg.Info.IsFromMe,
		msg.Message.GetAudioMessage() != nil)

	audio := msg.Message.GetAudioMessage()
	if audio == nil {
		return
	}
	chatJID := msg.Info.Chat.String()
	tracked := h.isTracked(msg)
	log.Printf("audio msg: chat=%s from=%s isFromMe=%v tracked=%v secs=%d",
		chatJID, msg.Info.Sender, msg.Info.IsFromMe, tracked, audio.GetSeconds())
	if msg.Info.IsFromMe {
		return
	}
	if !tracked {
		return
	}
	go h.transcribeAndReply(msg, audio)
}

func (h *Handler) recordChat(msg *events.Message) {
	if h.Chats == nil {
		return
	}
	chat := msg.Info.Chat
	entry := chats.Entry{
		JID:      chat.String(),
		Kind:     jidKind(chat),
		LastSeen: msg.Info.Timestamp,
	}
	// Prefer group subject for groups, push-name of the peer for DMs.
	if entry.Kind == "group" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if info, err := h.Client.GetGroupInfo(ctx, chat); err == nil {
			entry.Name = info.Name
		}
		cancel()
	} else if !msg.Info.IsFromMe && msg.Info.PushName != "" {
		entry.Name = msg.Info.PushName
	}
	if entry.Name == "" {
		entry.Name = chat.User
	}
	if err := h.Chats.Upsert(entry); err != nil {
		log.Printf("chat cache upsert: %v", err)
	}
}

func (h *Handler) onHistorySync(evt *events.HistorySync) {
	if h.Chats == nil || evt.Data == nil {
		return
	}
	convs := evt.Data.GetConversations()
	log.Printf("HistorySync chunk: type=%s conversations=%d progress=%d%%",
		evt.Data.GetSyncType(), len(convs), evt.Data.GetProgress())
	for _, conv := range convs {
		id := conv.GetID()
		if id == "" {
			continue
		}
		jid, err := types.ParseJID(id)
		if err != nil {
			continue
		}
		name := conv.GetName()
		if name == "" {
			name = conv.GetDisplayName()
		}
		if name == "" {
			name = jid.User
		}
		var ts time.Time
		if v := conv.GetConversationTimestamp(); v > 0 {
			ts = time.Unix(int64(v), 0)
		}
		_ = h.Chats.Upsert(chats.Entry{
			JID:      jid.String(),
			Name:     name,
			Kind:     jidKind(jid),
			LastSeen: ts,
		})
	}
}

// isTracked checks if the chat is tracked, treating LID and phone JID as
// aliases for the same person. WhatsApp's newer "LID" identifiers mean a DM
// can arrive with chat=X@lid while the user toggled X@s.whatsapp.net (or vice
// versa). We try both the raw JID, the alt-addressing JID exposed on the
// event, and the LID/PN mapping in the store.
func (h *Handler) isTracked(msg *events.Message) bool {
	chat := msg.Info.Chat
	if h.Tracker.Has(chat.String()) {
		return true
	}
	if !msg.Info.SenderAlt.IsEmpty() && msg.Info.Chat == msg.Info.Sender {
		// DM: SenderAlt is the peer's alternative address.
		if h.Tracker.Has(msg.Info.SenderAlt.String()) {
			return true
		}
	}
	if !msg.Info.RecipientAlt.IsEmpty() {
		if h.Tracker.Has(msg.Info.RecipientAlt.String()) {
			return true
		}
	}
	if h.Client.Store.LIDs != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		switch chat.Server {
		case types.HiddenUserServer:
			if pn, err := h.Client.Store.LIDs.GetPNForLID(ctx, chat); err == nil && !pn.IsEmpty() {
				if h.Tracker.Has(pn.String()) {
					return true
				}
			}
		case types.DefaultUserServer:
			if lid, err := h.Client.Store.LIDs.GetLIDForPN(ctx, chat); err == nil && !lid.IsEmpty() {
				if h.Tracker.Has(lid.String()) {
					return true
				}
			}
		}
	}
	return false
}

func jidKind(j types.JID) string {
	switch j.Server {
	case types.GroupServer:
		return "group"
	case types.DefaultUserServer:
		return "dm"
	case types.HiddenUserServer:
		return "lid"
	default:
		return j.Server
	}
}

func (h *Handler) transcribeAndReply(msg *events.Message, audio *waE2E.AudioMessage) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("voice msg in %s from %s (%ds)", msg.Info.Chat, msg.Info.Sender, audio.GetSeconds())

	data, err := h.Client.Download(ctx, audio)
	if err != nil {
		log.Printf("download failed: %v", err)
		h.sendReply(ctx, msg, fmt.Sprintf("[transcription error: %v]", err))
		return
	}
	text, err := h.Transcriber.Run(ctx, data)
	if err != nil {
		log.Printf("transcribe failed: %v", err)
		h.sendReply(ctx, msg, fmt.Sprintf("[transcription error: %v]", err))
		return
	}
	if text == "" {
		text = "[no speech detected]"
	}
	h.sendReply(ctx, msg, text)
}

func (h *Handler) sendReply(ctx context.Context, src *events.Message, text string) {
	reply := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("[транскрипція]: " + text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(src.Info.ID),
				Participant:   proto.String(src.Info.Sender.String()),
				QuotedMessage: src.Message,
			},
		},
	}
	if _, err := h.Client.SendMessage(ctx, src.Info.Chat, reply); err != nil {
		log.Printf("send reply failed: %v", err)
	}
}
