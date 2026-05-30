package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	appconfig "vimo-chat/internal/config"
)

func TestFirstSubmittedUserMessageStaysVisible(t *testing.T) {
	m := NewModel()
	m.showSplash = false
	m.config.Providers[appconfig.ProviderOpenAI] = appconfig.ProviderConfig{
		APIKey: "test-key",
		Model:  "gpt-4.1-mini",
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)

	m.textarea.SetValue("first")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	visible := terminalVisibleLines(m.View(), 8)
	if !strings.Contains(visible, "first") {
		t.Fatalf("first submitted message should remain visible; visible output:\n%s", visible)
	}
}

func TestViewFitsWindowHeightAfterResize(t *testing.T) {
	m := NewModel()
	m.showSplash = false

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)

	if got, want := lineCountForTest(m.View()), 8; got > want {
		t.Fatalf("view rendered %d lines, want at most %d", got, want)
	}
}

func TestViewFitsWindowHeightWithMenuOpen(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)
	m = submitInput(t, m, "/provider")

	if got, want := lineCountForTest(m.View()), 8; got > want {
		t.Fatalf("view rendered %d lines with menu open, want at most %d", got, want)
	}
}

func TestSlashOpensCommandPalette(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.ready = true
	m.windowWidth = 80

	m = typeRunes(t, m, "/")

	view := m.View()
	if !strings.Contains(view, "/provider") || !strings.Contains(view, "Connect provider") {
		t.Fatalf("slash command palette missing provider command:\n%s", view)
	}
	if !strings.Contains(view, "/model") || !strings.Contains(view, "Switch model") {
		t.Fatalf("slash command palette missing model command:\n%s", view)
	}
}

func TestCommandPaletteHasPaddingAndBottomRule(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.ready = true
	m.windowWidth = 40

	m = typeRunes(t, m, "/")

	palette := commandPaletteView(m)
	lines := strings.Split(palette, "\n")
	if len(lines) < len(commandOptions)+1 {
		t.Fatalf("palette rendered %d lines, want commands plus bottom rule:\n%s", len(lines), palette)
	}
	if !strings.HasPrefix(lines[0], " ") {
		t.Fatalf("first command line has no left padding: %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "─") {
		t.Fatalf("last palette line should be a bottom rule: %q", lines[len(lines)-1])
	}
}

func TestProviderCommandShowsPopup(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.ready = true
	m.windowWidth = 80

	m = submitInput(t, m, "/provider")

	view := m.View()
	if !strings.Contains(view, "Connect a provider") || !strings.Contains(view, "esc") {
		t.Fatalf("provider popup missing title/escape hint:\n%s", view)
	}
	if !strings.Contains(view, "OpenAI") || !strings.Contains(view, "ZAI") || !strings.Contains(view, "Deepseek") {
		t.Fatalf("provider popup missing provider options:\n%s", view)
	}
}

func TestProviderPopupLinesFillDialogWidth(t *testing.T) {
	dialog := renderPopup("Connect a provider", providerMenuRows(), 1, 80)
	lines := strings.Split(dialog, "\n")
	wantWidth := lipgloss.Width(lines[0])

	for i, line := range lines {
		if got := lipgloss.Width(line); got != wantWidth {
			t.Fatalf("line %d width = %d, want %d:\n%s", i, got, wantWidth, dialog)
		}
	}
}

func TestEscClosesPopup(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.ready = true
	m.windowWidth = 80

	m = submitInput(t, m, "/provider")
	m = pressKey(t, m, tea.KeyEsc)

	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want normal", m.mode)
	}
	if view := m.View(); strings.Contains(view, "Connect a provider") {
		t.Fatalf("provider popup remained visible after Esc:\n%s", view)
	}
}

func TestConfigCommandRedactsAPIKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewModelWithConfigPath(path)
	m.showSplash = false
	m.config.Providers[appconfig.ProviderOpenAI] = appconfig.ProviderConfig{
		APIKey: "sk-secret",
		Model:  "gpt-4.1-mini",
	}

	m = submitInput(t, m, "/config")

	if got := len(m.messages); got != 0 {
		t.Fatalf("config command added %d messages, want none", got)
	}

	view := renderMessage(m.messages) + "\n" + statusBar(m)
	if strings.Contains(view, "sk-secret") {
		t.Fatalf("config output leaked API key:\n%s", view)
	}
	if !strings.Contains(view, "key set") {
		t.Fatalf("config output should show redacted key state in status:\n%s", view)
	}
}

func TestMessageWithoutAPIKeyPromptsForConfig(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false

	m = submitInput(t, m, "hello")

	if got := len(m.messages); got != 0 {
		t.Fatalf("message count = %d, want no config prompt in conversation", got)
	}
	if status := statusBar(m); !strings.Contains(status, "API key missing") {
		t.Fatalf("status = %q, want API key missing prompt", status)
	}
}

func TestKeyCommandPersistsCurrentProviderKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewModelWithConfigPath(path)
	m.showSplash = false
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	m = submitInput(t, m, "/key")
	m = typeRunes(t, m, "sk-openai")

	if got := m.textarea.Value(); got != "" {
		t.Fatalf("textarea value = %q, want key input to stay inside dialog", got)
	}
	if view := m.View(); !strings.Contains(view, "sk-openai") {
		t.Fatalf("key dialog did not render typed key:\n%s", view)
	}

	m = pressKey(t, m, tea.KeyEnter)

	if got := len(m.messages); got != 0 {
		t.Fatalf("key command added %d messages, want none", got)
	}
	if got := m.config.Provider(appconfig.ProviderOpenAI).APIKey; got != "sk-openai" {
		t.Fatalf("in-memory key = %q, want saved key", got)
	}

	loaded, err := appconfig.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := loaded.Provider(appconfig.ProviderOpenAI).APIKey; got != "sk-openai" {
		t.Fatalf("persisted key = %q, want saved key", got)
	}
}

func TestProviderMenuPersistsSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewModelWithConfigPath(path)
	m.showSplash = false

	m = submitInput(t, m, "/provider")
	m = pressKey(t, m, tea.KeyDown)
	m = pressKey(t, m, tea.KeyEnter)

	if got := len(m.messages); got != 0 {
		t.Fatalf("provider command added %d messages, want none", got)
	}
	if got := m.config.ActiveProvider; got != appconfig.ProviderZAI {
		t.Fatalf("active provider = %q, want %q", got, appconfig.ProviderZAI)
	}
	if m.mode != modeKeyInput {
		t.Fatalf("mode = %v, want key input after selecting provider without key", m.mode)
	}

	loaded, err := appconfig.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := loaded.ActiveProvider; got != appconfig.ProviderZAI {
		t.Fatalf("persisted active provider = %q, want %q", got, appconfig.ProviderZAI)
	}
}

func TestProviderSelectionWithExistingKeyClosesPopup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewModelWithConfigPath(path)
	m.showSplash = false
	m.config.SetProviderConfig(appconfig.ProviderZAI, appconfig.ProviderConfig{
		APIKey: "zai-key",
		Model:  "glm-5.1",
	})

	m = submitInput(t, m, "/provider")
	m = pressKey(t, m, tea.KeyDown)
	m = pressKey(t, m, tea.KeyEnter)

	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want normal when selected provider already has a key", m.mode)
	}
}

func TestModelMenuPersistsActiveProviderModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewModelWithConfigPath(path)
	m.showSplash = false
	m.config.ActiveProvider = appconfig.ProviderDeepseek

	m = submitInput(t, m, "/model")
	m = pressKey(t, m, tea.KeyDown)
	m = pressKey(t, m, tea.KeyEnter)

	if got := len(m.messages); got != 0 {
		t.Fatalf("model command added %d messages, want none", got)
	}
	if got := m.config.Provider(appconfig.ProviderDeepseek).Model; got != "deepseek-reasoner" {
		t.Fatalf("deepseek model = %q, want deepseek-reasoner", got)
	}

	loaded, err := appconfig.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := loaded.Provider(appconfig.ProviderDeepseek).Model; got != "deepseek-reasoner" {
		t.Fatalf("persisted model = %q, want deepseek-reasoner", got)
	}
}

func TestStatusBarShowsProviderModelAndRedactedKeyState(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.windowWidth = 80
	m.config.ActiveProvider = appconfig.ProviderZAI
	m.config.Providers[appconfig.ProviderZAI] = appconfig.ProviderConfig{
		APIKey: "zai-secret",
		Model:  "glm-4.5",
	}

	status := statusBar(m)
	if !strings.Contains(status, "ZAI") || !strings.Contains(status, "glm-4.5") || !strings.Contains(status, "key set") {
		t.Fatalf("status bar missing config summary: %q", status)
	}
	if strings.Contains(status, "zai-secret") {
		t.Fatalf("status bar leaked API key: %q", status)
	}
}

func TestStatusBarFitsWindowWidth(t *testing.T) {
	m := NewModelWithConfigPath(filepath.Join(t.TempDir(), "config.json"))
	m.showSplash = false
	m.windowWidth = 24

	if got := lipgloss.Width(statusBar(m)); got > m.windowWidth {
		t.Fatalf("status width = %d, want at most %d: %q", got, m.windowWidth, statusBar(m))
	}
}

func submitInput(t *testing.T, m Model, input string) Model {
	t.Helper()
	m.textarea.SetValue(input)
	return pressKey(t, m, tea.KeyEnter)
}

func pressKey(t *testing.T, m Model, key tea.KeyType) Model {
	t.Helper()
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	return model
}

func typeRunes(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model, ok := updated.(Model)
		if !ok {
			t.Fatalf("Update returned %T, want Model", updated)
		}
		m = model
	}
	return m
}

func terminalVisibleLines(view string, height int) string {
	lines := strings.Split(view, "\n")
	if len(lines) <= height {
		return view
	}
	return strings.Join(lines[len(lines)-height:], "\n")
}

func lineCountForTest(s string) int {
	return len(strings.Split(s, "\n"))
}
