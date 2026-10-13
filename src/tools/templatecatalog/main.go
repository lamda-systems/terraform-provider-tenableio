// Command templatecatalog exports the definitions of Tenable.io's
// Tenable-provided scan templates, credential types and plugin families to a
// JSON catalogue.
//
// The catalogue is what the typed attributes of the scan template resources
// are generated from: the docs describe only a handful of the settings each
// template accepts, while GET /editor/{type}/templates/{uuid} describes all of
// them. It is read-only -- every call is a GET -- and it reads definitions
// that Tenable ships, not anything a tenant has created.
//
// A few inputs are populated from the tenant rather than by Tenable (the folder
// and scanner pickers, target fields, ownership and permissions). Their options
// and defaults are removed before anything is written, and the ids removed are
// listed in the output under "redacted_inputs". Review the file before sharing
// it all the same; "review" in the output points at anything the heuristics
// below thought looked tenant-specific.
//
// Usage, from src/:
//
//	TENABLEIO_ACCESS_KEY=... TENABLEIO_SECRET_KEY=... \
//	  go run ./tools/templatecatalog -out template-catalog.json
//
// TENABLEIO_BASE_URL, TENABLEIO_PROXY_AUTH_HEADER and TENABLEIO_PROXY_AUTH_VALUE
// are honoured the same way the provider honours them.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

// formatVersion is bumped whenever the shape of the catalogue changes, so a
// generator can refuse a file it does not understand.
const formatVersion = 1

// droppedKeys are removed wherever they appear: they describe who owns or may
// use an object, which is tenant data and of no use to a schema.
var droppedKeys = map[string]bool{
	"acls":             true,
	"owner":            true,
	"owner_id":         true,
	"owner_uuid":       true,
	"shared":           true,
	"user_permissions": true,
}

// tenantInputs are inputs whose options or default Tenable.io fills in from
// the tenant's own objects. Their definitions are kept; their values are not.
var tenantInputs = map[string]bool{
	"acls":                true,
	"agent_group_id":      true,
	"asset_lists":         true,
	"emails":              true,
	"file_targets":        true,
	"folder_id":           true,
	"scanner_id":          true,
	"tag_targets":         true,
	"target_network_uuid": true,
	"text_targets":        true,
}

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

type catalogue struct {
	Format          int                          `json:"format"`
	GeneratedAt     string                       `json:"generated_at"`
	TemplateLists   map[string][]client.Template `json:"template_lists"`
	TemplateDetails map[string]map[string]any    `json:"template_details"`
	CredentialTypes any                          `json:"credential_types,omitempty"`
	PluginFamilies  any                          `json:"plugin_families,omitempty"`
	RedactedInputs  []string                     `json:"redacted_inputs"`
	Review          []string                     `json:"review"`
	Errors          map[string]string            `json:"errors,omitempty"`
}

func main() {
	out := flag.String("out", "template-catalog.json", "file to write the catalogue to")
	typesFlag := flag.String("types", "policy,scan", "comma-separated template lists to export")
	flag.Parse()

	if err := run(*out, strings.Split(*typesFlag, ",")); err != nil {
		fmt.Fprintln(os.Stderr, "templatecatalog:", err)
		os.Exit(1)
	}
}

func run(out string, templateTypes []string) error {
	accessKey, secretKey := os.Getenv("TENABLEIO_ACCESS_KEY"), os.Getenv("TENABLEIO_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		return errors.New("set TENABLEIO_ACCESS_KEY and TENABLEIO_SECRET_KEY")
	}
	c := client.New(accessKey, secretKey, os.Getenv("TENABLEIO_BASE_URL"),
		os.Getenv("TENABLEIO_PROXY_AUTH_HEADER"), os.Getenv("TENABLEIO_PROXY_AUTH_VALUE"), "templatecatalog")
	ctx := context.Background()

	cat := catalogue{
		Format:          formatVersion,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		TemplateLists:   map[string][]client.Template{},
		TemplateDetails: map[string]map[string]any{},
		Errors:          map[string]string{},
	}
	clean := newSanitizer()

	for _, raw := range templateTypes {
		templateType := client.TemplateType(strings.TrimSpace(raw))
		list, err := withRetry(func() (*client.TemplatesListResponse, error) {
			return c.ListTemplates(ctx, templateType)
		})
		if err != nil {
			return fmt.Errorf("listing %s templates: %w", templateType, err)
		}
		cat.TemplateLists[string(templateType)] = list.Templates
		cat.TemplateDetails[string(templateType)] = map[string]any{}
		for _, t := range list.Templates {
			clean.known[t.UUID] = true
		}

		for _, t := range list.Templates {
			fmt.Fprintf(os.Stderr, "%s/%s\n", templateType, t.Name)
			details, err := getJSON(ctx, c, fmt.Sprintf("/editor/%s/templates/%s", templateType, t.UUID))
			if err != nil {
				// One unreadable template should not cost the rest.
				cat.Errors[fmt.Sprintf("%s/%s", templateType, t.Name)] = err.Error()
				continue
			}
			cat.TemplateDetails[string(templateType)][t.Name] =
				clean.clean(details, fmt.Sprintf("%s/%s", templateType, t.Name))
		}
	}

	for path, field := range map[string]*any{
		"/credentials/types": &cat.CredentialTypes,
		"/plugins/families":  &cat.PluginFamilies,
	} {
		fmt.Fprintln(os.Stderr, path)
		v, err := getJSON(ctx, c, path)
		if err != nil {
			cat.Errors[path] = err.Error()
			continue
		}
		*field = clean.clean(v, path)
	}

	cat.RedactedInputs = clean.sortedRedacted()
	cat.Review = clean.review
	if len(cat.Errors) == 0 {
		cat.Errors = nil
	}

	data, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding catalogue: %w", err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Fprintf(os.Stderr, "\nwrote %s: %d redacted input(s), %d item(s) to review, %d error(s)\n",
		out, len(cat.RedactedInputs), len(cat.Review), len(cat.Errors))
	for _, item := range cat.Review {
		fmt.Fprintln(os.Stderr, "  review:", item)
	}
	return nil
}

func getJSON(ctx context.Context, c *client.Client, path string) (any, error) {
	return withRetry(func() (any, error) {
		var v any
		err := c.Get(ctx, path, &v)
		return v, err
	})
}

// withRetry retries on rate limiting and server errors. A full export is a few
// dozen calls in a row, which is exactly what trips a 429.
func withRetry[T any](call func() (T, error)) (T, error) {
	delay := 2 * time.Second
	for attempt := 1; ; attempt++ {
		v, err := call()
		var apiErr *client.APIError
		retryable := errors.As(err, &apiErr) &&
			(apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500)
		if !retryable || attempt == 5 {
			return v, err
		}
		time.Sleep(delay)
		delay *= 2
	}
}

// sanitizer strips tenant data from a decoded JSON document and collects what
// it removed and what it is unsure about.
type sanitizer struct {
	// known holds the uuids of Tenable's own templates, which are expected in
	// the output and are not worth flagging.
	known    map[string]bool
	redacted map[string]bool
	review   []string
}

func newSanitizer() *sanitizer {
	return &sanitizer{known: map[string]bool{}, redacted: map[string]bool{}}
}

func (s *sanitizer) clean(v any, where string) any {
	switch node := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(node))
		for key, value := range node {
			if droppedKeys[key] {
				continue
			}
			out[key] = s.clean(value, where+"."+key)
		}
		if id, ok := out["id"].(string); ok && tenantInputs[id] {
			delete(out, "default")
			delete(out, "options")
			out["redacted"] = true
			s.redacted[id] = true
		}
		return out
	case []any:
		out := make([]any, len(node))
		for i, value := range node {
			out[i] = s.clean(value, fmt.Sprintf("%s[%d]", where, i))
		}
		return out
	case string:
		// Tenable's own definitions carry neither addresses nor object ids
		// as option values, so either one is worth a human look.
		if !s.known[node] && (emailPattern.MatchString(node) || uuidPattern.MatchString(node)) {
			s.review = append(s.review, where)
		}
		return node
	default:
		return node
	}
}

func (s *sanitizer) sortedRedacted() []string {
	ids := make([]string, 0, len(s.redacted))
	for id := range s.redacted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
