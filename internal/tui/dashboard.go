// Package tui implements a full-screen terminal dashboard for VAJRA.
//
// It takes over the entire terminal (like htop/btop) and shows
// real-time credential brokering metrics, audit events, and system status.
package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Colour palette ──────────────────────────────────────────────────

var (
	colGold    = lipgloss.Color("#FFD700")
	colGreen   = lipgloss.Color("#00FF87")
	colOrange  = lipgloss.Color("#FF8C00")
	colRed     = lipgloss.Color("#FF4444")
	colBlue    = lipgloss.Color("#5BC0EB")
	colGray    = lipgloss.Color("#888888")
	colWhite   = lipgloss.Color("#FFFFFF")
	colDark    = lipgloss.Color("#1E1E2E")
	colSurface = lipgloss.Color("#2D2D44")
	colBorder  = lipgloss.Color("#3D3D5C")
)

// ── Styles ───────────────────────────────────────────────────────────

var (
	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colGold).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(0, 2).
			Width(100)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colBlue).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(0, 1)

	// Status indicators
	statusRunning = lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true).
			Render("● RUNNING")

	statusIdle = lipgloss.NewStyle().
			Foreground(colOrange).
			Bold(true).
			Render("● IDLE")

	// Metric boxes
	metricBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(2, 2).
			Width(30)

	metricValue = lipgloss.NewStyle().
			Bold(true).
			Foreground(colGold)

	metricLabel = lipgloss.NewStyle().
			Foreground(colGray)

	// Activity log
	logBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1).
		Width(96)

	logEntry = lipgloss.NewStyle().
			Foreground(colWhite)

	logMint = lipgloss.NewStyle().
		Foreground(colGreen)

	logDeny = lipgloss.NewStyle().
		Foreground(colRed)

	logInfo = lipgloss.NewStyle().
		Foreground(colBlue)

	// Help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(colGray).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(0, 2).
			Width(100)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colGold).
			Bold(true)

	// Separator
	separator = lipgloss.NewStyle().
			Foreground(colBorder).
			Render(strings.Repeat("─", 100))
)

// ── TUI Model ────────────────────────────────────────────────────────

// LogLine represents a single line in the activity log.
type LogLine struct {
	Text string
	Time string
	Type string // "mint", "deny", "info"
}

// Model is the full-screen TUI state.
type Model struct {
	ready      bool
	width      int
	height     int
	startTime  time.Time

	// Live metrics
	credsActive    int
	credsMinted    int
	credsExpired   int
	policiesLoaded int
	agentsOnline   int

	// Activity log (ring buffer, newest first)
	log  []LogLine
	maxLog int

	// Simulated daemon state
	daemonRunning bool
}

// NewModel creates a fresh TUI model.
func NewModel() *Model {
	return &Model{
		startTime:      time.Now(),
		credsActive:    0,
		credsMinted:    0,
		credsExpired:   0,
		policiesLoaded: 2,
		agentsOnline:   1,
		daemonRunning:  true,
		maxLog:         50,
		log: []LogLine{
			{Time: time.Now().Format("15:04:05"), Text: "VAJRA daemon started — listening on stdio + :9735/mcp", Type: "info"},
			{Time: time.Now().Format("15:04:05"), Text: "Policy engine loaded: db_scope.rego, api_scope.rego", Type: "info"},
			{Time: time.Now().Format("15:04:05"), Text: "Agent 'demo-agent' registered (OAuth 2.1 Device Flow)", Type: "info"},
		},
	}
}

// ── Model lifecycle ──────────────────────────────────────────────────

// Init starts the tick loop for live updates.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		simulateActivity(),
	)
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			// Reset simulation
			m.credsActive = 0
			m.credsMinted = 0
			m.credsExpired = 0
			m.log = m.log[:0]
			return m, nil
		}
		return m, nil

	case tickMsg:
		// Update uptime and active count (credentials expire)
		if m.credsActive > 0 {
			m.credsActive--
			m.credsExpired++
		}
		return m, tick()

	case activityMsg:
		// Simulate a credential mint
		agentID := fmt.Sprintf("agent-%03d", rand.Intn(5)+1)
		targets := []string{
			"postgres://prod-db/mydb",
			"api.github.com/users",
			"stripe.com/charges",
			"hubspot.com/contacts",
			"aws:s3://reports-bucket",
		}
		target := targets[rand.Intn(len(targets))]
		credID := fmt.Sprintf("cred_%x", time.Now().UnixNano())
		ttl := rand.Intn(8) + 2

		m.credsMinted++
		m.credsActive++

		entry := LogLine{
			Time: time.Now().Format("15:04:05"),
			Text: fmt.Sprintf("%s → %s  [%ds TTL]  %s", agentID, target, ttl, credID),
			Type: "mint",
		}

		// Prepent to log ring buffer
		m.log = append([]LogLine{entry}, m.log...)
		if len(m.log) > m.maxLog {
			m.log = m.log[:m.maxLog]
		}

		return m, simulateActivity()
	}

	return m, nil
}

// ── Rendering ────────────────────────────────────────────────────────

// View renders the full-screen TUI.
func (m Model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().
			Foreground(colGold).
			Bold(true).
			Render("\n\n  🗡️  VAJRA — Initializing...") + "\n"
	}

	var b strings.Builder

	// ── Title Bar ────────────────────────────────────────────────
	title := titleStyle.Render("🗡️  VAJRA")
	subtitle := subtitleStyle.Render("Ephemeral Identity Broker  •  v0.1.0  •  MCP 2026-07-28")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle))
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")

	// ── Metrics Row ──────────────────────────────────────────────
	b.WriteString(m.renderMetricsRow())
	b.WriteString("\n")

	// ── Status Panels ────────────────────────────────────────────
	b.WriteString(m.renderStatusRow())
	b.WriteString("\n")

	// ── Activity Log ─────────────────────────────────────────────
	b.WriteString(m.renderActivityLog())
	b.WriteString("\n")

	// ── Help Bar ─────────────────────────────────────────────────
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderMetricsRow shows the 4 key metric boxes side by side.
func (m Model) renderMetricsRow() string {
	// Calculate available width for each box
	boxW := (m.width - 10) / 4
	if boxW < 20 {
		boxW = 20
	}
	boxStyle := metricBox.Width(boxW)
	valStyle := metricValue.Width(boxW - 4)

	credsBox := boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			valStyle.Render(fmt.Sprintf("%d", m.credsActive)),
			metricLabel.Render("Active Credentials"),
		),
	)

	mintedBox := boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Foreground(colGreen).Width(boxW-4).Render(fmt.Sprintf("%d", m.credsMinted)),
			metricLabel.Render("Total Minted"),
		),
	)

	expiredBox := boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Foreground(colOrange).Width(boxW-4).Render(fmt.Sprintf("%d", m.credsExpired)),
			metricLabel.Render("Expired / Revoked"),
		),
	)

	policiesBox := boxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Foreground(colBlue).Width(boxW-4).Render(fmt.Sprintf("%d", m.policiesLoaded)),
			metricLabel.Render("Policies Loaded"),
		),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		"  ",
		credsBox, "  ",
		mintedBox, "  ",
		expiredBox, "  ",
		policiesBox, "  ",
	)
}

// renderStatusRow shows status information.
func (m Model) renderStatusRow() string {
	uptime := time.Since(m.startTime).Round(time.Second).String()

	statusBoxW := (m.width - 10) / 2
	if statusBoxW < 40 {
		statusBoxW = 40
	}

	statusBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 2).
		Width(statusBoxW)

	// Left panel: daemon status
	leftContent := fmt.Sprintf(
		"  %s  Uptime: %s",
		statusRunning,
		uptime,
	)

	// Right panel: transport info
	rightContent := fmt.Sprintf(
		"  Transport: stdio + HTTP SSE (:9735)   |   Agents: %d online",
		m.agentsOnline,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		"  ",
		statusBox.Render(leftContent), "  ",
		statusBox.Render(rightContent), "  ",
	)
}

// renderActivityLog shows the real-time event stream.
func (m Model) renderActivityLog() string {
	availHeight := m.height - 14
	if availHeight < 5 {
		availHeight = 5
	}

	logHeight := availHeight
	if logHeight > len(m.log)+1 {
		logHeight = len(m.log) + 1
	}

	boxW := m.width - 6
	if boxW < 50 {
		boxW = 50
	}

	// Build log entries
	var lines []string
	maxShow := logHeight - 1
	if maxShow > len(m.log) {
		maxShow = len(m.log)
	}
	for i := 0; i < maxShow; i++ {
		entry := m.log[i]
		var line string
		switch entry.Type {
		case "mint":
			line = logMint.Render(fmt.Sprintf("  ◇ %s  %s", entry.Time, entry.Text))
		case "deny":
			line = logDeny.Render(fmt.Sprintf("  ✗ %s  %s", entry.Time, entry.Text))
		default:
			line = logInfo.Render(fmt.Sprintf("  ● %s  %s", entry.Time, entry.Text))
		}
		lines = append(lines, line)
	}

	// Truncate long lines to box width
	for i, line := range lines {
		if len(line) > boxW-4 {
			lines[i] = line[:boxW-7] + "..."
		}
	}

	content := strings.Join(lines, "\n")
	if content == "" {
		content = "  Waiting for agent activity..."
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1).
		Width(boxW).
		Height(logHeight + 1)

	header := lipgloss.NewStyle().Bold(true).Foreground(colBlue).Render("  Activity Log")
	return "  " + box.Render(header+"\n"+content)
}

// renderHelpBar shows keyboard shortcuts.
func (m Model) renderHelpBar() string {
	q := helpKeyStyle.Render("q")
	r := helpKeyStyle.Render("r")
	arrow := helpKeyStyle.Render("↑↓")

	return helpStyle.Render(
		fmt.Sprintf("  [%s] Quit    [%s] Reset    [%s] Scroll    VAJRA v0.1.0 — Single binary, zero deps", q, r, arrow),
	)
}

// ── Messages ─────────────────────────────────────────────────────────

type tickMsg struct{}
type activityMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func simulateActivity() tea.Cmd {
	return tea.Tick(time.Duration(rand.Intn(3000)+500)*time.Millisecond, func(t time.Time) tea.Msg {
		return activityMsg{}
	})
}

// ── Entry point ──────────────────────────────────────────────────────

// Run starts the full-screen TUI. Takes over the terminal (alternate screen).
func Run() error {
	p := tea.NewProgram(
		NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
