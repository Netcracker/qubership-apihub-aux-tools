// Package config loads the YAML config file and overlays environment variables.
// Final precedence (applied in main): flag > env > config file > default.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Env variable names.
const (
	EnvApihubAPIKey        = "APIHUB_API_KEY"
	EnvGitlabToken         = "GITLAB_TOKEN"
	EnvDdlGitlabToken      = "DDL_GITLAB_TOKEN"
	EnvCommentsGitlabToken = "COMMENTS_GITLAB_TOKEN"
)

// Source types.
const (
	SourceFile   = "file"
	SourceGitlab = "gitlab"
)

// Source describes where an input (DDL folder or comments workbook) comes from.
type Source struct {
	Type   string `yaml:"type"`
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
	Path   string `yaml:"path"`
	Token  string `yaml:"token"`
}

// Config mirrors the YAML config file.
type Config struct {
	Apihub struct {
		URL       string `yaml:"url"`
		APIKey    string `yaml:"apiKey"`
		PackageID string `yaml:"packageId"`
	} `yaml:"apihub"`
	DDLSource      Source `yaml:"ddlSource"`
	CommentsSource Source `yaml:"commentsSource"`
	Output         struct {
		Dir string `yaml:"dir"`
	} `yaml:"output"`
	Publish struct {
		VersionLabels []string `yaml:"versionLabels"`
		Timeout       string   `yaml:"timeout"` // Go duration string, default 15m
	} `yaml:"publish"`
	Groups struct {
		Enabled             *bool  `yaml:"enabled"` // default true
		DescriptionTemplate string `yaml:"descriptionTemplate"`
	} `yaml:"groups"`
	Analytics struct {
		DataTypeChangeRegex string `yaml:"dataTypeChangeRegex"`
	} `yaml:"analytics"`
}

// Load reads the YAML file. A missing file is an error only when explicit is
// true (the user passed --config themselves).
func Load(path string, explicit bool) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// ApplyEnv fills secrets from the environment when the config file left them
// empty. Per-source token env vars beat the shared GITLAB_TOKEN.
func (c *Config) ApplyEnv() {
	if v := os.Getenv(EnvApihubAPIKey); v != "" && c.Apihub.APIKey == "" {
		c.Apihub.APIKey = v
	}
	shared := os.Getenv(EnvGitlabToken)
	if v := os.Getenv(EnvDdlGitlabToken); v != "" {
		c.DDLSource.Token = v
	} else if shared != "" && c.DDLSource.Token == "" {
		c.DDLSource.Token = shared
	}
	if v := os.Getenv(EnvCommentsGitlabToken); v != "" {
		c.CommentsSource.Token = v
	} else if shared != "" && c.CommentsSource.Token == "" {
		c.CommentsSource.Token = shared
	}
}

// ApplyDefaults fills unset optional values.
func (c *Config) ApplyDefaults() {
	if c.Output.Dir == "" {
		c.Output.Dir = "./ddl-import-out"
	}
	if c.Publish.Timeout == "" {
		c.Publish.Timeout = "15m"
	}
	if c.Groups.DescriptionTemplate == "" {
		c.Groups.DescriptionTemplate = "Tables of the %s domain"
	}
	if c.DDLSource.Type == "" {
		c.DDLSource.Type = SourceFile
	}
	if c.CommentsSource.Type == "" {
		c.CommentsSource.Type = SourceFile
	}
	if c.DDLSource.Branch == "" {
		c.DDLSource.Branch = "main"
	}
	if c.CommentsSource.Branch == "" {
		c.CommentsSource.Branch = "main"
	}
}

// PublishTimeout parses the publish poll timeout.
func (c *Config) PublishTimeout() (time.Duration, error) {
	d, err := time.ParseDuration(c.Publish.Timeout)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid publish timeout %q", c.Publish.Timeout)
	}
	return d, nil
}

// GroupsEnabled resolves the groups.enabled default (true).
func (c *Config) GroupsEnabled() bool {
	return c.Groups.Enabled == nil || *c.Groups.Enabled
}

// ValidateSource checks a source definition; name is "ddlSource"/"commentsSource".
func ValidateSource(name string, s Source) error {
	switch s.Type {
	case SourceFile:
		if s.Path == "" {
			return fmt.Errorf("%s: path is required for type=file", name)
		}
	case SourceGitlab:
		if s.Repo == "" || s.Path == "" {
			return fmt.Errorf("%s: repo and path are required for type=gitlab", name)
		}
	default:
		return fmt.Errorf("%s: unknown type %q (want file or gitlab)", name, s.Type)
	}
	return nil
}
