package config

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadFromConf(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"/home/me/.config/md2c/md2c.conf": "MD2C_BASE_URL=https://from-home.example\nMD2C_USER=home-user\nMD2C_TOKEN=\"home-token\"\nMD2C_PREFIX=banner\n",
	}
	cfg, err := Load(Sources{
		Home: "/home/me",
		Read: func(path string) ([]byte, error) {
			if body, ok := files[path]; ok {
				return []byte(body), nil
			}
			return nil, errors.New("not found")
		},
		Getenv: func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://from-home.example" || cfg.User != "home-user" || cfg.Token != "home-token" || cfg.Prefix != "banner" || cfg.Auth != "basic" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadIgnoresProcessEnv(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"/home/me/.config/md2c/md2c.conf": "MD2C_BASE_URL=https://file.example\nMD2C_USER=file-user\nMD2C_TOKEN=file-token\n",
	}
	cfg, err := Load(Sources{
		Home: "/home/me",
		Read: func(path string) ([]byte, error) {
			if body, ok := files[path]; ok {
				return []byte(body), nil
			}
			return nil, errors.New("not found")
		},
		Getenv: func(key string) string {
			if key == "MD2C_TOKEN" {
				return "shell-token"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "file-token" {
		t.Fatalf("token %q (config must win, env ignored)", cfg.Token)
	}
}

func TestLoadAuthBearer(t *testing.T) {
	t.Parallel()
	cfg, err := Load(Sources{
		Home: "/home/me",
		Read: func(path string) ([]byte, error) {
			if path == "/home/me/.config/md2c/md2c.conf" {
				return []byte("MD2C_BASE_URL=https://wiki.example\nMD2C_USER=me\nMD2C_TOKEN=pat\nMD2C_AUTH=bearer\n"), nil
			}
			return nil, errors.New("not found")
		},
		Getenv: func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth != "bearer" {
		t.Fatalf("auth %q", cfg.Auth)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()
	_, err := Load(Sources{
		Home:   "/home/me",
		Read:   func(string) ([]byte, error) { return nil, errors.New("not found") },
		Getenv: func(string) string { return "" },
	})
	if err == nil || !strings.Contains(err.Error(), "keine Config") {
		t.Fatalf("err %v", err)
	}
}

func TestLoadIncomplete(t *testing.T) {
	t.Parallel()
	_, err := Load(Sources{
		Home: "/home/me",
		Read: func(path string) ([]byte, error) {
			if path == "/home/me/.config/md2c/md2c.conf" {
				return []byte("MD2C_BASE_URL=https://x.example\n"), nil
			}
			return nil, errors.New("not found")
		},
		Getenv: func(string) string { return "" },
	})
	if err == nil || !strings.Contains(err.Error(), "MD2C_USER") || !strings.Contains(err.Error(), "MD2C_TOKEN") {
		t.Fatalf("err %v", err)
	}
}

func TestParseEnvFile(t *testing.T) {
	t.Parallel()
	got := parseEnvFile([]byte("# comment\nexport MD2C_USER=me\nMD2C_TOKEN='ab=c'\n\nNOEQUALS\n"))
	if got["MD2C_USER"] != "me" {
		t.Fatalf("user %q", got["MD2C_USER"])
	}
	if got["MD2C_TOKEN"] != "ab=c" {
		t.Fatalf("token %q", got["MD2C_TOKEN"])
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()
	home := "/home/me"
	if got := ExpandPath("~/.config/md2c/md2c.conf", home); got != "/home/me/.config/md2c/md2c.conf" {
		t.Fatalf("tilde %q", got)
	}
	if got := ExpandPath("/abs/md2c.conf", home); got != "/abs/md2c.conf" {
		t.Fatalf("abs %q", got)
	}
	if got := ExpandPath("", home); got != "" {
		t.Fatalf("empty %q", got)
	}
}

func TestLoadExplicitPath(t *testing.T) {
	t.Parallel()
	cfg, err := Load(Sources{
		Home: "/home/me",
		Path: "~/.config/md2c/other.conf",
		Read: func(path string) ([]byte, error) {
			if path != "/home/me/.config/md2c/other.conf" {
				t.Fatalf("path %q", path)
			}
			return []byte("MD2C_BASE_URL=https://other.example\nMD2C_USER=u\nMD2C_TOKEN=t\n"), nil
		},
		Getenv: func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://other.example" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestPathXDG(t *testing.T) {
	t.Parallel()
	got := Path("/home/me", func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return "/xdg"
		}
		return ""
	})
	if got != "/xdg/md2c/md2c.conf" {
		t.Fatalf("path %q", got)
	}
}
