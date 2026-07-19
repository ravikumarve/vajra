// Package dashboard serves the VAJRA web dashboard.
//
// The dashboard is a single-binary embedded web UI that mirrors the TUI
// in the browser. It reuses the same design tokens as the landing page
// and streams live credential data via SSE.
package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/vajra/vajra/internal/audit"
	"github.com/vajra/vajra/internal/auth"
	"github.com/vajra/vajra/internal/credential"
	"github.com/vajra/vajra/internal/policy"
)

// ── Embedded templates ─────────────────────────────────────────────

//go:embed templates/*.html
var templateFS embed.FS

// ── Template data types ────────────────────────────────────────────

// PageData is the data passed to every template.
type PageData struct {
	Title       string
	Version     string
	Uptime      string
	CredCount   int
	CredMinted  int
	CredExpired int
	PolicyCount int
	AgentCount  int
	Credentials []*credential.Credential
	AuditEntries []audit.Entry
	Agents      []*auth.AgentInfo
	Policies    []string
	ActiveTab   string
	DemoMode    bool
}

// ── SSE Event ──────────────────────────────────────────────────────

// SSEClient holds a channel for one connected browser tab.
type SSEClient struct {
	ID   string
	Ch   chan []byte
}

// SSEBroker manages all connected SSE clients.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
	nextID  int
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[string]*SSEClient),
	}
}

func (b *SSEBroker) Subscribe() *SSEClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := fmt.Sprintf("sse_%d", b.nextID)
	c := &SSEClient{ID: id, Ch: make(chan []byte, 64)}
	b.clients[id] = c
	return c
}

func (b *SSEBroker) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c, ok := b.clients[id]; ok {
		close(c.Ch)
		delete(b.clients, id)
	}
}

func (b *SSEBroker) Publish(data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, c := range b.clients {
		select {
		case c.Ch <- data:
		default:
			// drop slow clients
		}
	}
}

// ── Dashboard Server ───────────────────────────────────────────────

// Server serves the VAJRA web dashboard using embedded templates.
type Server struct {
	ring      *credential.RingBuffer
	auditLog  *audit.Logger
	registry  *auth.Registry
	policy    *policy.Engine
	sse       *SSEBroker
	templates *template.Template
	startTime time.Time
	demoMode  bool

	mu          sync.Mutex
	credMinted  int
	credExpired int
}

// NewServer creates a dashboard server wired to the daemon's subsystems.
func NewServer(
	ring *credential.RingBuffer,
	auditLog *audit.Logger,
	registry *auth.Registry,
	policy *policy.Engine,
) *Server {
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "--"
			}
			return t.Format("15:04:05")
		},
		"remaining": func(expiresAt time.Time) string {
			r := time.Until(expiresAt)
			if r <= 0 {
				return "0.0s"
			}
			return fmt.Sprintf("%.1fs", r.Seconds())
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			if d < time.Minute {
				return fmt.Sprintf("%.0fs ago", d.Seconds())
			}
			if d < time.Hour {
				return fmt.Sprintf("%.0fm ago", d.Minutes())
			}
			return fmt.Sprintf("%.0fh ago", d.Hours())
		},
	}

	tmpl := template.Must(
		template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"),
	)

	return &Server{
		ring:      ring,
		auditLog:  auditLog,
		registry:  registry,
		policy:    policy,
		sse:       NewSSEBroker(),
		templates: tmpl,
		startTime: time.Now(),
	}
}

// Handler returns the HTTP handler for the dashboard (mount at e.g. /dashboard).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard/", s.handlePage)
	mux.HandleFunc("/dashboard/api/creds", s.apiCredentials)
	mux.HandleFunc("/dashboard/api/audit", s.apiAudit)
	mux.HandleFunc("/dashboard/api/stats", s.apiStats)
	mux.HandleFunc("/dashboard/events", s.handleSSE)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

// PublishEvent is called by the daemon when a credential is minted, revoked, or expired.
func (s *Server) PublishEvent(eventType string, data interface{}) {
	s.mu.Lock()
	switch eventType {
	case "mint":
		s.credMinted++
	case "expire", "revoke":
		s.credExpired++
	}
	s.mu.Unlock()

	event, err := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": data,
		"ts":   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	s.sse.Publish(event)
}

// ── Demo / Simulation ──────────────────────────────────────────────

var demoAgentIDs = []string{"agent-alpha", "agent-beta", "agent-gamma", "agent-delta"}
var demoTargets = []string{
	"postgres://prod-db/customers",
	"api.stripe.com/v1/charges",
	"api.github.com/repos/vajra",
	"hubspot.com/contacts",
	"aws:s3://reports-bucket",
	"slack.com/api/conversations",
}
var demoScopes = []string{"read:email", "read:subs", "write:status", "read:users", "admin:reports"}

// EnableDemoMode registers demo agents and starts live simulation.
func (s *Server) EnableDemoMode() {
	s.demoMode = true
	for _, id := range demoAgentIDs {
		_ = s.registry.Register(id, "Demo "+id, "", "oauth", []string{"default"})
	}
	s.SeedDemoData()
	s.StartSimulation()
}

// SeedDemoData populates the ring buffer and audit log with initial demo entries.
func (s *Server) SeedDemoData() {
	now := time.Now()

	// Seed 5-8 initial credentials into the ring buffer
	count := 5 + rand.Intn(4)
	for i := 0; i < count; i++ {
		agent := demoAgentIDs[rand.Intn(len(demoAgentIDs))]
		target := demoTargets[rand.Intn(len(demoTargets))]
		scope := demoScopes[rand.Intn(len(demoScopes))]
		ttl := time.Duration(3+rand.Intn(8)) * time.Second

		c := credential.NewCredential(agent, target, scope, ttl)
		c.CreatedAt = now.Add(-time.Duration(rand.Intn(10)) * time.Second)
		c.ExpiresAt = c.CreatedAt.Add(ttl)
		s.ring.Insert(c)
		s.mu.Lock()
		s.credMinted++
		s.mu.Unlock()

		// Log to audit
		if s.auditLog != nil {
			s.auditLog.Log("mint", agent, c.ID, target, scope, ttl.String(), "allow", "Demo credential", nil)
		}
		s.PublishEvent("mint", c)
	}

	// Seed some expired entries in audit (so audit page has history)
	expiredAgents := []string{"agent-alpha", "agent-beta"}
	expiredTargets := []string{"stripe.com/old", "db_archive"}
	for i := 0; i < 5; i++ {
		a := expiredAgents[rand.Intn(len(expiredAgents))]
		t := expiredTargets[rand.Intn(len(expiredTargets))]
		if s.auditLog != nil {
			s.auditLog.Log("mint", a, fmt.Sprintf("cred_hist_%d", i), t, "read", "5s", "allow", "Historical demo", nil)
		}
	}
	// An expired/denied entry
	if s.auditLog != nil {
		s.auditLog.Log("deny", "agent-gamma", "", "api.stripe.com/admin", "admin:*", "", "deny", "Policy violation: scope not permitted", nil)
		s.auditLog.Log("revoke", "agent-delta", "cred_rev_1", "api.github.com", "write", "3s", "revoke", "Manual revocation via vajra revoke", nil)
	}

	log.Printf("[dashboard] demo: seeded %d credentials, %d agents", count, len(demoAgentIDs))
}

// StartSimulation spawns a goroutine that mints ~1 credential every 2-5s.
func (s *Server) StartSimulation() {
	go func() {
		for {
			delay := time.Duration(2000+rand.Intn(3000)) * time.Millisecond
			time.Sleep(delay)

			agent := demoAgentIDs[rand.Intn(len(demoAgentIDs))]
			target := demoTargets[rand.Intn(len(demoTargets))]
			scope := demoScopes[rand.Intn(len(demoScopes))]
			ttl := time.Duration(2+rand.Intn(6)) * time.Second

			c := credential.NewCredential(agent, target, scope, ttl)
			s.ring.Insert(c)

			s.mu.Lock()
			s.credMinted++
			s.mu.Unlock()

			if s.auditLog != nil {
				s.auditLog.Log("mint", agent, c.ID, target, scope, ttl.String(), "allow", "Live demo credential", nil)
			}
			s.PublishEvent("mint", c)
		}
	}()
	log.Printf("[dashboard] demo: simulation started (1 cred every 2-5s)")
}

// ── Page handlers ──────────────────────────────────────────────────

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	// Determine which tab from the URL path
	tab := "overview"
	switch r.URL.Path {
	case "/dashboard/audit":
		tab = "audit"
	case "/dashboard/agents":
		tab = "agents"
	}

	s.mu.Lock()
	minted := s.credMinted
	expired := s.credExpired
	s.mu.Unlock()

	creds := s.ring.All()
	agents := s.registry.List()

	var auditEntries []audit.Entry
	if s.auditLog != nil {
		entries, err := s.auditLog.Query(50)
		if err == nil {
			auditEntries = entries
		}
	}
	if auditEntries == nil {
		auditEntries = []audit.Entry{}
	}

	policyNames := s.policy.List()
	if policyNames == nil {
		policyNames = []string{}
	}

	uptime := time.Since(s.startTime).Round(time.Second).String()

	data := PageData{
		Title:        "VAJRA · " + tab,
		Version:      "0.1.0",
		Uptime:       uptime,
		CredCount:    len(creds),
		CredMinted:   minted,
		CredExpired:  expired,
		PolicyCount:  len(policyNames),
		AgentCount:   len(agents),
		Credentials:  creds,
		AuditEntries: auditEntries,
		Agents:       agents,
		Policies:     policyNames,
		ActiveTab:    tab,
		DemoMode:     s.demoMode,
	}

	tmplName := tab // "overview", "audit", or "agents"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, tmplName, data); err != nil {
		logError("template", err)
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
	}
}

// ── JSON API handlers ──────────────────────────────────────────────

func (s *Server) apiCredentials(w http.ResponseWriter, r *http.Request) {
	creds := s.ring.All()
	writeJSON(w, creds)
}

func (s *Server) apiAudit(w http.ResponseWriter, r *http.Request) {
	var entries []audit.Entry
	if s.auditLog != nil {
		e, err := s.auditLog.Query(100)
		if err == nil {
			entries = e
		}
	}
	if entries == nil {
		entries = []audit.Entry{}
	}
	writeJSON(w, entries)
}

func (s *Server) apiStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	minted := s.credMinted
	expired := s.credExpired
	s.mu.Unlock()

	creds := s.ring.All()
	agents := s.registry.Count()

	writeJSON(w, map[string]interface{}{
		"cred_active":  len(creds),
		"cred_minted":  minted,
		"cred_expired": expired,
		"agents":       agents,
		"policies":     len(s.policy.List()),
		"uptime_sec":   time.Since(s.startTime).Seconds(),
	})
}

// ── SSE handler ────────────────────────────────────────────────────

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := s.sse.Subscribe()
	defer s.sse.Unsubscribe(client.ID)

	// Send initial keepalive
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-client.Ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			// Keepalive tick — triggers JS to refresh stats
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// ── Health ─────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "vajra-dashboard"})
}

// ── Helpers ────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func logError(context string, err error) {
	log.Printf("[dashboard] %s: %v", context, err)
}
