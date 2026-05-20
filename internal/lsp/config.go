package lsp

import "github.com/howznguyen/knowns/internal/models"

type Config struct {
	Languages map[string]LanguageConfig
}

type LanguageConfig struct {
	Enabled  *bool
	Binary   string
	Version  string
	Settings map[string]any
}

func ConfigFromProject(project *models.Project) Config {
	if project == nil || project.Settings.LSP == nil {
		return Config{}
	}
	languages := make(map[string]LanguageConfig, len(project.Settings.LSP.Languages))
	for name, entry := range project.Settings.LSP.Languages {
		languages[name] = LanguageConfig{
			Enabled:  entry.Enabled,
			Binary:   entry.Binary,
			Version:  entry.Version,
			Settings: entry.Settings,
		}
	}
	return Config{Languages: languages}
}

func (c Config) Enabled(language string) bool {
	entry, ok := c.Languages[language]
	if !ok || entry.Enabled == nil {
		return true
	}
	return *entry.Enabled
}

func (c Config) BinaryOverride(language string) string {
	entry, ok := c.Languages[language]
	if !ok {
		return ""
	}
	return entry.Binary
}

func (c Config) VersionOverride(language string) string {
	entry, ok := c.Languages[language]
	if !ok {
		return ""
	}
	return entry.Version
}

func (c Config) LanguageSettings(language string) map[string]any {
	entry, ok := c.Languages[language]
	if !ok {
		return nil
	}
	return entry.Settings
}

func (c Config) IsDisabled(language string) bool {
	entry, ok := c.Languages[language]
	if !ok {
		return false
	}
	if entry.Enabled == nil {
		return false
	}
	return !*entry.Enabled
}
