package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds Confluence connection settings from md2c.conf.
type Config struct {
	BaseURL string
	User    string
	Token   string
	Prefix  string
	Auth    string
}

// Sources are lookup hooks so tests can avoid the real filesystem.
type Sources struct {
	Getenv func(string) string
	Read   func(path string) ([]byte, error)
	Home   string
	Path   string
}

// Path is ~/.config/md2c/md2c.conf (or $XDG_CONFIG_HOME/md2c/md2c.conf).
func Path(home string, getenv func(string) string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	dir := filepath.Join(home, ".config", "md2c")
	if xdg := getenv("XDG_CONFIG_HOME"); xdg != "" {
		dir = filepath.Join(xdg, "md2c")
	}
	return filepath.Join(dir, "md2c.conf")
}

// ExpandPath resolves ~ and ~/ against home. Other paths are returned trimmed.
func ExpandPath(raw, home string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "~" {
		return home
	}
	if strings.HasPrefix(raw, "~/") {
		return filepath.Join(home, raw[2:])
	}
	return raw
}

// Load reads md2c.conf. Default path is ~/.config/md2c/md2c.conf unless Sources.Path
// is set (CLI -config). It does not use process environment for credentials.
func Load(src Sources) (Config, error) {
	if src.Getenv == nil {
		src.Getenv = os.Getenv
	}
	if src.Read == nil {
		src.Read = os.ReadFile
	}

	path := ExpandPath(src.Path, src.Home)
	if path == "" {
		path = Path(src.Home, src.Getenv)
	}
	raw, err := src.Read(path)
	if err != nil {
		return Config{}, fmt.Errorf("keine Config %s — anlegen mit MD2C_BASE_URL, MD2C_USER, MD2C_TOKEN (siehe md2c.conf.example)", path)
	}

	kv := parseEnvFile(raw)
	cfg := Config{
		BaseURL: strings.TrimSpace(kv["MD2C_BASE_URL"]),
		User:    firstNonEmpty(kv["MD2C_USER"], kv["MD2CUSER"]),
		Token:   firstNonEmpty(kv["MD2C_TOKEN"], kv["MD2C_PASS"], kv["MD2CPASS"]),
		Prefix:  strings.TrimSpace(kv["MD2C_PREFIX"]),
		Auth:    resolveAuth(kv["MD2C_AUTH"]),
	}

	var missing []string
	if cfg.BaseURL == "" {
		missing = append(missing, "MD2C_BASE_URL")
	}
	if cfg.User == "" {
		missing = append(missing, "MD2C_USER")
	}
	if cfg.Token == "" {
		missing = append(missing, "MD2C_TOKEN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("%s: es fehlen %s", path, strings.Join(missing, ", "))
	}
	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func resolveAuth(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bearer", "pat", "token":
		return "bearer"
	default:
		return "basic"
	}
}

func parseEnvFile(raw []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if q := value[0]; (q == '"' || q == '\'') && value[len(value)-1] == q {
				value = value[1 : len(value)-1]
			}
		}
		if key != "" {
			out[key] = value
		}
	}
	return out
}
