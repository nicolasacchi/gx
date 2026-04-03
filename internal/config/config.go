package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Project struct {
	Token string `toml:"token,omitempty"`
	Owner string `toml:"owner"`
	Repo  string `toml:"repo"`
}

type Config struct {
	DefaultProject string              `toml:"default_project,omitempty"`
	Projects       map[string]*Project `toml:"projects,omitempty"`
}

type Credentials struct {
	Token string
	Owner string
	Repo  string
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gx", "config.toml"), nil
}

func loadConfigFile() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfigFile(cfg *Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func resolveProject(cfg *Config, projectFlag string) *Project {
	if cfg == nil {
		return nil
	}
	if projectFlag != "" && cfg.Projects != nil {
		if p, ok := cfg.Projects[projectFlag]; ok {
			return p
		}
		return nil
	}
	if cfg.DefaultProject != "" && cfg.Projects != nil {
		if p, ok := cfg.Projects[cfg.DefaultProject]; ok {
			return p
		}
	}
	return nil
}

// ghAuthToken reads the token from gh CLI.
func ghAuthToken() string {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LoadCredentials resolves credentials: flag > env > gh auth > config file.
func LoadCredentials(tokenFlag, ownerFlag, repoFlag, projectFlag string) (*Credentials, error) {
	creds := &Credentials{}

	// Config file for defaults
	cfg, _ := loadConfigFile()
	if p := resolveProject(cfg, projectFlag); p != nil {
		creds.Token = p.Token
		creds.Owner = p.Owner
		creds.Repo = p.Repo
	}

	// gh auth token (fallback)
	if creds.Token == "" {
		creds.Token = ghAuthToken()
	}

	// Env vars override config
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		creds.Token = v
	}
	if v := os.Getenv("GX_OWNER"); v != "" {
		creds.Owner = v
	}
	if v := os.Getenv("GX_REPO"); v != "" {
		creds.Repo = v
	}

	// Flags override everything
	if tokenFlag != "" {
		creds.Token = tokenFlag
	}
	if ownerFlag != "" {
		creds.Owner = ownerFlag
	}
	if repoFlag != "" {
		creds.Repo = repoFlag
	}

	if creds.Token == "" {
		return nil, fmt.Errorf("token required: use --token flag, GITHUB_TOKEN env var, gh auth login, or 'gx config add'")
	}
	if creds.Owner == "" {
		return nil, fmt.Errorf("owner required: use --owner flag, GX_OWNER env var, or 'gx config add'")
	}
	if creds.Repo == "" {
		return nil, fmt.Errorf("repo required: use --repo flag, GX_REPO env var, or 'gx config add'")
	}

	return creds, nil
}

func AddProject(name, owner, repo string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		cfg = &Config{}
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]*Project)
	}
	cfg.Projects[name] = &Project{Owner: owner, Repo: repo}
	if cfg.DefaultProject == "" {
		cfg.DefaultProject = name
	}
	return saveConfigFile(cfg)
}

func RemoveProject(name string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		return fmt.Errorf("no config file found")
	}
	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("project %q not found", name)
	}
	delete(cfg.Projects, name)
	if cfg.DefaultProject == name {
		cfg.DefaultProject = ""
		for k := range cfg.Projects {
			cfg.DefaultProject = k
			break
		}
	}
	return saveConfigFile(cfg)
}

func SetDefaultProject(name string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		return fmt.Errorf("no config file found")
	}
	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("project %q not found", name)
	}
	cfg.DefaultProject = name
	return saveConfigFile(cfg)
}

func ListProjects() (*Config, error) {
	return loadConfigFile()
}

func MaskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:8] + "***" + token[len(token)-4:]
}
