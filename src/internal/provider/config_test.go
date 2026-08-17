package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// clearEnv blanks every variable resolveSettings consults so a test only sees
// what it sets itself, regardless of the developer's shell.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, suffix := range []string{envAccessKey, envSecretKey, envBaseURL, envProxyAuthHeader, envProxyAuthValue} {
		t.Setenv(envPrefix+"_"+suffix, "")
		t.Setenv("TENABLEIO_EU_"+suffix, "")
	}
}

// baseEnv sets credentials that satisfy the required-key checks so a test can
// focus on the setting it actually exercises.
func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envPrefix+"_"+envAccessKey, "shared-access")
	t.Setenv(envPrefix+"_"+envSecretKey, "shared-secret")
}

// Note: these tests call t.Setenv and so must not call t.Parallel.

func TestResolveSettingsSources(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		config TenableioProviderModel
		want   providerSettings
	}{
		{
			name: "unprefixed environment only",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY": "shared-access",
				"TENABLEIO_SECRET_KEY": "shared-secret",
			},
			want: providerSettings{accessKey: "shared-access", secretKey: "shared-secret"},
		},
		{
			name: "prefixed wins over unprefixed",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":    "shared-access",
				"TENABLEIO_SECRET_KEY":    "shared-secret",
				"TENABLEIO_EU_ACCESS_KEY": "eu-access",
				"TENABLEIO_EU_SECRET_KEY": "eu-secret",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")},
			want:   providerSettings{accessKey: "eu-access", secretKey: "eu-secret"},
		},
		{
			name: "per variable fallback mixes prefixed and unprefixed",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":           "shared-access",
				"TENABLEIO_SECRET_KEY":           "shared-secret",
				"TENABLEIO_EU_PROXY_AUTH_HEADER": "Proxy-Authorization",
				"TENABLEIO_EU_PROXY_AUTH_VALUE":  "eu-token",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")},
			want: providerSettings{
				accessKey:       "shared-access",
				secretKey:       "shared-secret",
				proxyAuthHeader: "Proxy-Authorization",
				proxyAuthValue:  "eu-token",
			},
		},
		{
			name: "prefixed provider with no proxy variables of its own inherits the shared pair",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":        "shared-access",
				"TENABLEIO_SECRET_KEY":        "shared-secret",
				"TENABLEIO_PROXY_AUTH_HEADER": "Proxy-Authorization",
				"TENABLEIO_PROXY_AUTH_VALUE":  "shared-token",
				"TENABLEIO_EU_ACCESS_KEY":     "eu-access",
				"TENABLEIO_EU_SECRET_KEY":     "eu-secret",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")},
			want: providerSettings{
				accessKey:       "eu-access",
				secretKey:       "eu-secret",
				proxyAuthHeader: "Proxy-Authorization",
				proxyAuthValue:  "shared-token",
			},
		},
		{
			name: "prefixed provider overriding only the proxy value keeps the shared header",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":          "shared-access",
				"TENABLEIO_SECRET_KEY":          "shared-secret",
				"TENABLEIO_PROXY_AUTH_HEADER":   "Proxy-Authorization",
				"TENABLEIO_PROXY_AUTH_VALUE":    "shared-token",
				"TENABLEIO_EU_PROXY_AUTH_VALUE": "eu-token",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")},
			want: providerSettings{
				accessKey:       "shared-access",
				secretKey:       "shared-secret",
				proxyAuthHeader: "Proxy-Authorization",
				proxyAuthValue:  "eu-token",
			},
		},
		{
			name: "attribute beats both prefixed and unprefixed",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":    "shared-access",
				"TENABLEIO_SECRET_KEY":    "shared-secret",
				"TENABLEIO_EU_ACCESS_KEY": "eu-access",
				"TENABLEIO_EU_BASE_URL":   "https://eu.env.example.com",
			},
			config: TenableioProviderModel{
				Prefix:    types.StringValue("TENABLEIO_EU"),
				AccessKey: types.StringValue("literal-access"),
				BaseURL:   types.StringValue("https://eu.hcl.example.com"),
			},
			want: providerSettings{
				accessKey: "literal-access",
				secretKey: "shared-secret",
				baseURL:   "https://eu.hcl.example.com",
			},
		},
		{
			name: "trailing underscore in prefix is trimmed",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":    "shared-access",
				"TENABLEIO_SECRET_KEY":    "shared-secret",
				"TENABLEIO_EU_ACCESS_KEY": "eu-access",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU_")},
			want:   providerSettings{accessKey: "eu-access", secretKey: "shared-secret"},
		},
		{
			name: "empty prefixed variable falls through to unprefixed",
			env: map[string]string{
				"TENABLEIO_ACCESS_KEY":    "shared-access",
				"TENABLEIO_SECRET_KEY":    "shared-secret",
				"TENABLEIO_EU_ACCESS_KEY": "",
			},
			config: TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")},
			want:   providerSettings{accessKey: "shared-access", secretKey: "shared-secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, diags := resolveSettings(tt.config)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags.Errors())
			}
			if got != tt.want {
				t.Errorf("resolveSettings() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveSettingsErrors(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		config      TenableioProviderModel
		withBaseEnv bool
		wantSummary string
	}{
		{
			name:        "missing access key",
			config:      TenableioProviderModel{SecretKey: types.StringValue("secret")},
			wantSummary: "Missing API Access Key",
		},
		{
			name:        "missing secret key",
			config:      TenableioProviderModel{AccessKey: types.StringValue("access")},
			wantSummary: "Missing API Secret Key",
		},
		{
			name:        "proxy value without header",
			withBaseEnv: true,
			config:      TenableioProviderModel{ProxyAuthValue: types.StringValue("token")},
			wantSummary: "Missing Proxy Auth Header",
		},
		{
			name:        "proxy header without value",
			withBaseEnv: true,
			config:      TenableioProviderModel{ProxyAuthHeader: types.StringValue("Proxy-Authorization")},
			wantSummary: "Missing Proxy Auth Value",
		},
		{
			name:        "proxy header from env without value",
			withBaseEnv: true,
			env:         map[string]string{"TENABLEIO_PROXY_AUTH_HEADER": "Proxy-Authorization"},
			wantSummary: "Missing Proxy Auth Value",
		},
		{
			name:        "invalid prefix",
			withBaseEnv: true,
			config:      TenableioProviderModel{Prefix: types.StringValue("tenable-eu")},
			wantSummary: "Invalid Provider Prefix",
		},
		{
			name:        "unknown access key",
			withBaseEnv: true,
			config:      TenableioProviderModel{AccessKey: types.StringUnknown()},
			wantSummary: "Unknown Provider Configuration Value",
		},
		{
			name:        "unknown prefix",
			withBaseEnv: true,
			config:      TenableioProviderModel{Prefix: types.StringUnknown()},
			wantSummary: "Unknown Provider Configuration Value",
		},
		{
			name:        "unknown proxy auth value",
			withBaseEnv: true,
			config:      TenableioProviderModel{ProxyAuthValue: types.StringUnknown()},
			wantSummary: "Unknown Provider Configuration Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			if tt.withBaseEnv {
				baseEnv(t)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, diags := resolveSettings(tt.config)
			if !diags.HasError() {
				t.Fatalf("expected an error diagnostic, got none")
			}

			for _, d := range diags.Errors() {
				if d.Summary() == tt.wantSummary {
					return
				}
			}
			t.Errorf("expected diagnostic %q, got %v", tt.wantSummary, diags.Errors())
		})
	}
}

// A prefixed provider's diagnostics should name the variable that instance
// actually reads, not the shared one.
func TestResolveSettingsErrorNamesPrefixedVariable(t *testing.T) {
	clearEnv(t)

	_, diags := resolveSettings(TenableioProviderModel{Prefix: types.StringValue("TENABLEIO_EU")})
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic, got none")
	}

	for _, d := range diags.Errors() {
		if d.Summary() == "Missing API Access Key" {
			if want := "TENABLEIO_EU_ACCESS_KEY"; !strings.Contains(d.Detail(), want) {
				t.Errorf("detail %q does not mention %q", d.Detail(), want)
			}
			return
		}
	}
	t.Errorf("expected a Missing API Access Key diagnostic, got %v", diags.Errors())
}

// An aliased provider that must not use the inherited proxy opts out by
// setting both attributes to the empty string; a configured attribute always
// wins over the environment, including when it is empty.
func TestResolveSettingsProxyOptOut(t *testing.T) {
	clearEnv(t)
	baseEnv(t)
	t.Setenv(envPrefix+"_"+envProxyAuthHeader, "Proxy-Authorization")
	t.Setenv(envPrefix+"_"+envProxyAuthValue, "shared-token")

	got, diags := resolveSettings(TenableioProviderModel{
		Prefix:          types.StringValue("TENABLEIO_EU"),
		ProxyAuthHeader: types.StringValue(""),
		ProxyAuthValue:  types.StringValue(""),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
	if got.proxyAuthHeader != "" || got.proxyAuthValue != "" {
		t.Errorf("expected the inherited proxy to be opted out, got header=%q value=%q",
			got.proxyAuthHeader, got.proxyAuthValue)
	}
}

// Opting out of only one half of the pair is rejected: the other half is still
// inherited, which would send the shared token under an empty header name.
func TestResolveSettingsPartialProxyOptOutIsRejected(t *testing.T) {
	clearEnv(t)
	baseEnv(t)
	t.Setenv(envPrefix+"_"+envProxyAuthHeader, "Proxy-Authorization")
	t.Setenv(envPrefix+"_"+envProxyAuthValue, "shared-token")

	_, diags := resolveSettings(TenableioProviderModel{
		Prefix:          types.StringValue("TENABLEIO_EU"),
		ProxyAuthHeader: types.StringValue(""),
	})
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic, got none")
	}
	for _, d := range diags.Errors() {
		if d.Summary() == "Missing Proxy Auth Header" {
			return
		}
	}
	t.Errorf("expected a Missing Proxy Auth Header diagnostic, got %v", diags.Errors())
}

// Terraform gives each provider block its own configuration: attributes set on
// the default provider are invisible to an aliased one. Setting the header in
// HCL while the value comes from the environment therefore breaks every alias
// that does not repeat the header, because the alias inherits only the value.
func TestResolveSettingsAliasDoesNotInheritHCLProxyHeader(t *testing.T) {
	clearEnv(t)
	baseEnv(t)
	// The shared token lives in the environment; the header name does not,
	// because it was written into the default provider block instead.
	t.Setenv(envPrefix+"_"+envProxyAuthValue, "shared-token")

	// The aliased provider varies only by base_url, exactly as an operator
	// would write it.
	_, diags := resolveSettings(TenableioProviderModel{
		BaseURL: types.StringValue("https://eu.cloud.tenable.com"),
	})
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic, got none")
	}
	for _, d := range diags.Errors() {
		if d.Summary() == "Missing Proxy Auth Header" {
			return
		}
	}
	t.Errorf("expected a Missing Proxy Auth Header diagnostic, got %v", diags.Errors())
}

// Neither proxy setting configured is valid: the header is simply not sent.
func TestResolveSettingsProxyOmittedEntirely(t *testing.T) {
	clearEnv(t)
	baseEnv(t)

	got, diags := resolveSettings(TenableioProviderModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
	if got.proxyAuthHeader != "" || got.proxyAuthValue != "" {
		t.Errorf("expected no proxy settings, got header=%q value=%q", got.proxyAuthHeader, got.proxyAuthValue)
	}
}
