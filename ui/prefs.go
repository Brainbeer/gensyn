// ui/prefs.go

package ui

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Prefs holds user preferences persisted to ~/.config/gensyn/prefs.json
type Prefs struct {
	Theme            string  `json:"theme"`             // "dark" or "light"
	FontSize         float32 `json:"font_size"`         // 9–14
	DefaultOperation string  `json:"default_operation"` // emerge flag preset
	ClearOutput      bool    `json:"clear_output"`      // clear terminal on new command
	Editor           string  `json:"editor"`            // editor key, or "" for none
	CustomEditor     string  `json:"custom_editor"`     // path used when Editor == "custom"
}

// Current holds the active preferences for the running session.
var Current Prefs

func defaultPrefs() Prefs {
	return Prefs{
		Theme:            "dark",
		FontSize:         11,
		DefaultOperation: "No Flag",
		ClearOutput:      true,
		Editor:           "",
		CustomEditor:     "",
	}
}

func prefsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gensyn", "prefs.json"), nil
}

// LoadPrefs reads preferences from disk, returning defaults on any error.
func LoadPrefs() Prefs {
	path, err := prefsPath()
	if err != nil {
		return defaultPrefs()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultPrefs()
	}
	var p Prefs
	if err := json.Unmarshal(data, &p); err != nil {
		return defaultPrefs()
	}
	return p
}

// SavePrefs writes preferences to disk, creating the directory if needed.
func SavePrefs(p Prefs) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// editorMap maps the stored key to the executable name.
var editorMap = map[string]string{
	"mousepad":   "mousepad",
	"pluma":      "pluma",
	"kwrite":     "kwrite",
	"kate":       "kate",
	"gedit":      "gedit",
	"geany":      "geany",
	"xed":        "xed",
	"featherpad": "featherpad",
	"subl":       "subl",
	"code":       "code",
	"atom":       "atom",
}

// EditorExecutable returns the executable to launch for editing, or "" if none is set.
func EditorExecutable() string {
	if Current.Editor == "custom" {
		return Current.CustomEditor
	}
	return editorMap[Current.Editor]
}

// forcedTheme wraps the default Fyne theme, overriding color variant and text size.
type forcedTheme struct {
	variant  fyne.ThemeVariant
	fontSize float32
}

func (f *forcedTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, f.variant)
}

func (f *forcedTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (f *forcedTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (f *forcedTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return f.fontSize
	}
	return theme.DefaultTheme().Size(name)
}

// ApplyTheme pushes the current preferences into the Fyne app's theme.
func ApplyTheme(a fyne.App, p Prefs) {
	variant := theme.VariantDark
	if p.Theme == "light" {
		variant = theme.VariantLight
	}
	a.Settings().SetTheme(&forcedTheme{variant: variant, fontSize: p.FontSize})
}
