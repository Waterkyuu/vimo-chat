package tui

import (
	"fmt"

	appconfig "vimo-chat/internal/config"
)

type providerOption struct {
	id    appconfig.Provider
	label string
}

type commandOption struct {
	name        string
	description string
}

var commandOptions = []commandOption{
	{name: "/provider", description: "Connect provider"},
	{name: "/model", description: "Switch model"},
	{name: "/key", description: "Set API key"},
	{name: "/config", description: "Show config"},
	{name: "/compact", description: "Compress context"},
	{name: "/new", description: "Start new chat"},
	{name: "/clear", description: "Clear context"},
}

var providerOptions = []providerOption{
	{id: appconfig.ProviderOpenAI, label: "OpenAI"},
	{id: appconfig.ProviderZAI, label: "ZAI"},
	{id: appconfig.ProviderDeepseek, label: "Deepseek"},
}

var modelOptions = map[appconfig.Provider][]string{
	appconfig.ProviderOpenAI:   {"gpt-5.5", "gpt-5.4", "gpt-5.1"},
	appconfig.ProviderZAI:      {"glm-5.1", "glm-4.7", "glm-4.6"},
	appconfig.ProviderDeepseek: {"deepseek-v4-pro", "deepseek-v4-flash"},
}

func (m *Model) handleCommand(input string) bool {
	switch input {
	case "/provider":
		m.mode = modeProviderMenu
		m.menuIndex = m.currentProviderIndex()
		m.textarea.Reset()
		m.refreshLayout()
		return true
	case "/model":
		m.mode = modeModelMenu
		m.menuIndex = m.currentModelIndex()
		m.textarea.Reset()
		m.refreshLayout()
		return true
	case "/key":
		m.mode = modeKeyInput
		m.keyProvider = m.config.ActiveProvider
		m.keyInput = ""
		m.textarea.Reset()
		m.refreshLayout()
		return true
	case "/config":
		m.setNotice(m.configSummary())
		m.textarea.Reset()
		return true
	case "/new":
		m.messages = nil
		// Reset so the next sent message starts a fresh conversation ID.
		m.convID = ""
		m.assistantText = ""
		m.streaming = false
		m.textarea.Reset()
		m.refreshLayout()
		return true
	case "/clear":
		// Clear the working context (message history) but keep the current
		// conversation ID so prior extracted memories and summary stay archived.
		m.messages = nil
		m.assistantText = ""
		m.streaming = false
		m.textarea.Reset()
		m.refreshLayout()
		return true
	default:
		return false
	}
}

func (m *Model) saveAPIKey() {
	provider := m.keyProvider
	providerConfig := m.config.Provider(provider)
	providerConfig.APIKey = m.keyInput
	m.config.SetProviderConfig(provider, providerConfig)
	m.persistConfig()
	m.setNotice(fmt.Sprintf("API key saved for %s", providerLabel(provider)))
	m.keyInput = ""
	m.textarea.Reset()
	m.mode = modeNormal
	m.refreshLayout()
}

func (m *Model) selectProvider() {
	option := providerOptions[m.menuIndex]
	m.config.ActiveProvider = option.id
	m.persistConfig()
	m.setNotice(fmt.Sprintf("Provider switched to %s", option.label))
	if m.config.Provider(option.id).APIKey == "" {
		m.mode = modeKeyInput
		m.keyProvider = option.id
		m.keyInput = ""
	} else {
		m.mode = modeNormal
	}
	m.refreshLayout()
}

func (m *Model) selectModel() {
	provider := m.config.ActiveProvider
	models := currentModelOptions(provider)
	providerConfig := m.config.Provider(provider)
	providerConfig.Model = models[m.menuIndex]
	m.config.SetProviderConfig(provider, providerConfig)
	m.persistConfig()
	m.setNotice(fmt.Sprintf("%s model switched to %s", providerLabel(provider), providerConfig.Model))
	m.mode = modeNormal
	m.refreshLayout()
}

func (m *Model) moveMenu(delta int) {
	size := len(providerOptions)
	switch m.mode {
	case modeCommandPalette:
		size = len(commandOptions)
	case modeModelMenu:
		size = len(currentModelOptions(m.config.ActiveProvider))
	}
	if size == 0 {
		m.menuIndex = 0
		return
	}
	m.menuIndex = (m.menuIndex + delta + size) % size
}

func (m *Model) setNotice(content string) {
	m.notice = content
}

func (m *Model) persistConfig() {
	if m.configPath == "" {
		return
	}
	if err := appconfig.Save(m.configPath, m.config); err != nil {
		m.err = err
	}
}

func (m Model) activeProviderConfig() appconfig.ProviderConfig {
	return m.config.Provider(m.config.ActiveProvider)
}

func (m Model) currentProviderIndex() int {
	for i, option := range providerOptions {
		if option.id == m.config.ActiveProvider {
			return i
		}
	}
	return 0
}

func (m Model) currentModelIndex() int {
	current := m.config.Provider(m.config.ActiveProvider).Model
	for i, model := range currentModelOptions(m.config.ActiveProvider) {
		if model == current {
			return i
		}
	}
	return 0
}

func (m Model) configSummary() string {
	keyState := "key missing"
	if m.activeProviderConfig().APIKey != "" {
		keyState = "key set"
	}
	return fmt.Sprintf("%s | %s | %s", providerLabel(m.config.ActiveProvider), m.activeProviderConfig().Model, keyState)
}

func currentModelOptions(provider appconfig.Provider) []string {
	if models, ok := modelOptions[provider]; ok {
		return models
	}
	return modelOptions[appconfig.ProviderOpenAI]
}

func providerLabel(provider appconfig.Provider) string {
	for _, option := range providerOptions {
		if option.id == provider {
			return option.label
		}
	}
	return "OpenAI"
}
