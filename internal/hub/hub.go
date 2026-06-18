package hub

import (
	"errors"
	"fmt"
	"net"
	neturl "net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/blaxel-ai/mcp-hub/internal/smithery"
	"gopkg.in/yaml.v2"
)

type Hub struct {
	Repositories map[string]*Repository `yaml:"repositories"`
}

type PackageManager string

const (
	PackageManagerAPK PackageManager = "apk"
	PackageManagerAPT PackageManager = "apt"
)

type Repository struct {
	Repository      string                   `yaml:"repository" mandatory:"false"`
	Path            string                   `yaml:"path" mandatory:"false"`
	SmitheryPath    string                   `yaml:"smitheryPath" mandatory:"false" default:"smithery.yaml"`
	Smithery        *smithery.SmitheryConfig `yaml:"smithery" mandatory:"false"`
	Dockerfile      string                   `yaml:"dockerfile" mandatory:"false" default:"Dockerfile"`
	PackageManager  PackageManager           `yaml:"packageManager" mandatory:"false" default:"apk"`
	DoNotShow       []string                 `yaml:"doNotShow" mandatory:"false"`
	HasNPM          bool                     `yaml:"hasNPM" mandatory:"false" default:"true"`
	Branch          string                   `yaml:"branch" mandatory:"false" default:"main"`
	URL             string                   `yaml:"url" mandatory:"false"`
	Transport       string                   `yaml:"transport" mandatory:"false"`
	DisplayName     string                   `yaml:"displayName" mandatory:"true"`
	Icon            string                   `yaml:"icon" mandatory:"true"`
	Disabled        bool                     `yaml:"disabled" mandatory:"false" default:"false"`
	Description     string                   `yaml:"description" mandatory:"true"`
	LongDescription string                   `yaml:"longDescription" mandatory:"true"`
	Enterprise      bool                     `yaml:"enterprise" mandatory:"false" default:"false"`
	ComingSoon      bool                     `yaml:"comingSoon" mandatory:"false" default:"false"`
	Secrets         []string                 `yaml:"secrets" mandatory:"false"`
	HiddenSecrets   []string                 `yaml:"hiddenSecrets" mandatory:"false"`
	OAuth           *OAuth                   `yaml:"oauth" mandatory:"false"`
	HTTPUpstream    *HTTPUpstream            `yaml:"httpUpstream" mandatory:"false"`
	Integration     string                   `yaml:"integration" mandatory:"false"`
	Tags            []string                 `yaml:"tags"`
	Categories      []string                 `yaml:"categories"`
}

type HTTPUpstream struct {
	URL         string `yaml:"url"`
	AllowedPath string `yaml:"allowedPath"`
}

type OAuth struct {
	Type   string   `yaml:"type"`
	Scopes []string `yaml:"scopes"`
}

func (h *HTTPUpstream) ValidateWithDefaultValues() error {
	upstreamURL, err := h.parsedURL()
	if err != nil {
		return err
	}
	if h.AllowedPath == "" {
		h.AllowedPath = upstreamURL.Path
		if h.AllowedPath == "" {
			h.AllowedPath = "/mcp"
		}
	}
	if !strings.HasPrefix(h.AllowedPath, "/") {
		return fmt.Errorf("httpUpstream.allowedPath must start with /")
	}
	if h.AllowedPath == "/" {
		return fmt.Errorf("httpUpstream.allowedPath must not be /")
	}
	if strings.Contains(h.AllowedPath, "://") || strings.ContainsAny(h.AllowedPath, "?#") {
		return fmt.Errorf("httpUpstream.allowedPath must be a path without query or fragment")
	}
	return nil
}

func (h *HTTPUpstream) SuperGatewayArgs() ([]string, error) {
	if h == nil {
		return nil, nil
	}
	if err := h.ValidateWithDefaultValues(); err != nil {
		return nil, err
	}
	return []string{
		"--port", "80",
		"--transport", "http-stream",
		"--http-upstream", h.URL,
		"--http-upstream-path", h.AllowedPath,
		"--stdio",
	}, nil
}

func (h *HTTPUpstream) parsedURL() (*neturl.URL, error) {
	if h.URL == "" {
		return nil, fmt.Errorf("httpUpstream.url is required")
	}
	upstreamURL, err := neturl.Parse(h.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid httpUpstream.url: %w", err)
	}
	if upstreamURL.Scheme != "http" {
		return nil, fmt.Errorf("httpUpstream.url must use http scheme")
	}
	if upstreamURL.User != nil {
		return nil, fmt.Errorf("httpUpstream.url must not include userinfo")
	}
	if upstreamURL.RawQuery != "" || upstreamURL.Fragment != "" {
		return nil, fmt.Errorf("httpUpstream.url must not include query or fragment")
	}
	host := upstreamURL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("httpUpstream.url must include a host")
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("httpUpstream.url host must be loopback")
		}
	}
	return upstreamURL, nil
}

func (h *Hub) Read(path string) error {
	h.Repositories = make(map[string]*Repository)
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		yamlFile, err := os.ReadFile(filepath.Join(path, file.Name()))
		if err != nil {
			return err
		}

		var repo Repository
		if err := yaml.Unmarshal(yamlFile, &repo); err != nil {
			return err
		}

		// Use filename without extension as repository name
		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		h.Repositories[name] = &repo
	}
	return nil
}

// ValidateWithDefaultValues validates the hub and applies default values to empty fields
// This is useful to validate the hub before running the import command
func (h *Hub) ValidateWithDefaultValues() error {
	if h.Repositories == nil {
		return errors.New("repositories is required")
	}

	var errs []error

	for name, repository := range h.Repositories {
		// Use reflection to validate struct tags
		v := reflect.ValueOf(repository).Elem() // Get the element the pointer refers to
		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)

			// Check mandatory fields
			if mandatory, ok := field.Tag.Lookup("mandatory"); ok && mandatory == "true" {
				if value.IsZero() {
					errs = append(errs, fmt.Errorf("field %s is required in repository %s", field.Name, name))
				}
			}

			// Apply default values for empty fields
			if defaultVal, ok := field.Tag.Lookup("default"); ok && value.IsZero() {
				switch value.Kind() {
				case reflect.String:
					value.SetString(defaultVal)
				case reflect.Bool:
					value.SetBool(defaultVal == "true")
				}
			}
		}

		if repository.HTTPUpstream != nil {
			if err := repository.HTTPUpstream.ValidateWithDefaultValues(); err != nil {
				errs = append(errs, fmt.Errorf("repository %s: %w", name, err))
			}
			if repository.Transport == "" {
				repository.Transport = "http-stream"
			}
		}
	}

	return errors.Join(errs...)
}
