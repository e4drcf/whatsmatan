package web

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"sort"
	"time"

	"github.com/e4drcf/whatsmatan/internal/chats"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type Tracker interface {
	Has(jid string) bool
	All() []string
	Set(jid string, tracked bool) error
}

type ChatCache interface {
	All() []chats.Entry
}

type Server struct {
	Client  *whatsmeow.Client
	Tracker Tracker
	Chats   ChatCache
	Static  fs.FS // serves index.html etc.
}

type chatDTO struct {
	JID      string    `json:"jid"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"` // "group" | "dm" | "lid"
	Tracked  bool      `json:"tracked"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

type statusDTO struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"loggedIn"`
	JID       string `json:"jid"`
}

type trackedReq struct {
	JID     string `json:"jid"`
	Tracked bool   `json:"tracked"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chats", s.handleChats)
	mux.HandleFunc("/api/tracked", s.handleTracked)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.Handle("/", http.FileServer(http.FS(s.Static)))
	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var jid string
	if id := s.Client.Store.ID; id != nil {
		jid = id.String()
	}
	writeJSON(w, http.StatusOK, statusDTO{
		Connected: s.Client.IsConnected(),
		LoggedIn:  s.Client.IsLoggedIn(),
		JID:       jid,
	})
}

func (s *Server) handleChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	byJID := make(map[string]*chatDTO)

	// Live groups: source of truth for group names/membership.
	groups, err := s.Client.GetJoinedGroups(ctx)
	if err == nil {
		for _, g := range groups {
			jid := g.JID.String()
			byJID[jid] = &chatDTO{
				JID:     jid,
				Name:    g.Name,
				Kind:    "group",
				Tracked: s.Tracker.Has(jid),
			}
		}
	}

	// Cached chats observed via HistorySync + incoming messages.
	if s.Chats != nil {
		for _, e := range s.Chats.All() {
			if cur, ok := byJID[e.JID]; ok {
				if cur.LastSeen.Before(e.LastSeen) {
					cur.LastSeen = e.LastSeen
				}
				if cur.Name == "" {
					cur.Name = e.Name
				}
				continue
			}
			byJID[e.JID] = &chatDTO{
				JID:      e.JID,
				Name:     fallbackName(e.Name, e.JID),
				Kind:     e.Kind,
				Tracked:  s.Tracker.Has(e.JID),
				LastSeen: e.LastSeen,
			}
		}
	}

	// Address book contacts (only for DMs; groups already covered).
	if contacts, err := s.Client.Store.Contacts.GetAllContacts(ctx); err == nil {
		for jid, info := range contacts {
			if jid.Server != types.DefaultUserServer {
				continue
			}
			key := jid.String()
			if _, ok := byJID[key]; ok {
				continue
			}
			name := contactName(info, jid.User)
			byJID[key] = &chatDTO{
				JID:     key,
				Name:    name,
				Kind:    "dm",
				Tracked: s.Tracker.Has(key),
			}
		}
	}

	out := make([]chatDTO, 0, len(byJID))
	for _, c := range byJID {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		// Recent first; chats without a timestamp sink below those with one.
		ai, aj := out[i].LastSeen, out[j].LastSeen
		if ai.IsZero() != aj.IsZero() {
			return aj.IsZero()
		}
		if !ai.Equal(aj) {
			return ai.After(aj)
		}
		return out[i].Name < out[j].Name
	})
	writeJSON(w, http.StatusOK, out)
}

func fallbackName(name, jid string) string {
	if name != "" {
		return name
	}
	return jid
}

func contactName(info types.ContactInfo, user string) string {
	if info.FullName != "" {
		return info.FullName
	}
	if info.PushName != "" {
		return info.PushName
	}
	if info.FirstName != "" {
		return info.FirstName
	}
	if info.BusinessName != "" {
		return info.BusinessName
	}
	return user
}

func (s *Server) handleTracked(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Tracker.All())
	case http.MethodPost:
		var req trackedReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if req.JID == "" {
			writeErr(w, http.StatusBadRequest, errors.New("missing jid"))
			return
		}
		if _, err := types.ParseJID(req.JID); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := s.Tracker.Set(req.JID, req.Tracked); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
