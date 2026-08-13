package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	envPrefix = "TENABLEIO"

	envAccessKey       = "ACCESS_KEY"
	envSecretKey       = "SECRET_KEY"
	envBaseURL         = "BASE_URL"
	envProxyAuthHeader = "PROXY_AUTH_HEADER"
	envProxyAuthValue  = "PROXY_AUTH_VALUE"
)

// prefixPattern matches the environment variable name fragments a prefix is
// allowed to produce.
var prefixPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// providerSettings holds the fully resolved configuration for a single
// provider instance. Each aliased provider block resolves its own copy.
type providerSettings struct {
	accessKey       string
	secretKey       string
	baseURL         string
	proxyAuthHeader string
	proxyAuthValue  string
}

// resolveSettings resolves each setting independently from, in order: the
// configuration attribute, the prefixed environment variable (when prefix is
// set), then the unprefixed TENABLEIO_ variable. Falling back per variable
// lets an aliased provider override only its proxy credentials while sharing
// the base Tenable credentials.
func resolveSettings(config TenableioProviderModel) (providerSettings, diag.Diagnostics) {
	var diags diag.Diagnostics

	unknowns := []struct {
		name  string
		value types.String
	}{
		{"prefix", config.Prefix},
		{"access_key", config.AccessKey},
		{"secret_key", config.SecretKey},
		{"base_url", config.BaseURL},
		{"proxy_auth_header", config.ProxyAuthHeader},
		{"proxy_auth_value", config.ProxyAuthValue},
	}
	for _, u := range unknowns {
		if u.value.IsUnknown() {
			diags.AddAttributeError(
				path.Root(u.name),
				"Unknown Provider Configuration Value",
				fmt.Sprintf("The provider cannot create the Tenable.io API client because %q is not known until apply. "+
					"Provider configuration must be resolvable at plan time: set it to a literal value, or leave it unset "+
					"and supply it through an environment variable.", u.name),
			)
		}
	}
	if diags.HasError() {
		return providerSettings{}, diags
	}

	prefix := strings.TrimRight(config.Prefix.ValueString(), "_")
	if prefix != "" && !prefixPattern.MatchString(prefix) {
		diags.AddAttributeError(
			path.Root("prefix"),
			"Invalid Provider Prefix",
			fmt.Sprintf("The prefix %q cannot be used to build environment variable names. "+
				"A prefix must start with a letter or underscore and contain only letters, digits, and underscores, "+
				"for example %q.", config.Prefix.ValueString(), "TENABLEIO_EU"),
		)
		return providerSettings{}, diags
	}

	settings := providerSettings{
		accessKey:       resolve(config.AccessKey, prefix, envAccessKey),
		secretKey:       resolve(config.SecretKey, prefix, envSecretKey),
		baseURL:         resolve(config.BaseURL, prefix, envBaseURL),
		proxyAuthHeader: resolve(config.ProxyAuthHeader, prefix, envProxyAuthHeader),
		proxyAuthValue:  resolve(config.ProxyAuthValue, prefix, envProxyAuthValue),
	}

	if settings.accessKey == "" {
		diags.AddError(
			"Missing API Access Key",
			"The provider cannot create the Tenable.io API client because the access key is missing. "+
				"Set the access_key attribute in the provider configuration or the "+
				envVarName(prefix, envAccessKey)+" environment variable.",
		)
	}

	if settings.secretKey == "" {
		diags.AddError(
			"Missing API Secret Key",
			"The provider cannot create the Tenable.io API client because the secret key is missing. "+
				"Set the secret_key attribute in the provider configuration or the "+
				envVarName(prefix, envSecretKey)+" environment variable.",
		)
	}

	// The header name and its value are only meaningful together: a name
	// without a value sends an empty header to the proxy, and a value without
	// a name is dropped silently. Both can arrive from either source, so this
	// is checked on the resolved values rather than on the configuration.
	if settings.proxyAuthHeader == "" && settings.proxyAuthValue != "" {
		diags.AddError(
			"Missing Proxy Auth Header",
			"The provider was given a proxy auth value but no header name to send it under, so it would be silently dropped. "+
				"Set the proxy_auth_header attribute in the provider configuration or the "+
				envVarName(prefix, envProxyAuthHeader)+" environment variable.",
		)
	}

	if settings.proxyAuthHeader != "" && settings.proxyAuthValue == "" {
		diags.AddError(
			"Missing Proxy Auth Value",
			fmt.Sprintf("The provider was asked to send the %q header but no value for it, so the header would be sent empty. ",
				settings.proxyAuthHeader)+
				"Set the proxy_auth_value attribute in the provider configuration or the "+
				envVarName(prefix, envProxyAuthValue)+" environment variable.",
		)
	}

	if diags.HasError() {
		return providerSettings{}, diags
	}

	return settings, diags
}

// resolve returns the first of: the configured attribute, the prefixed
// environment variable, the unprefixed one. An environment variable that is
// set but empty is treated as unset so it falls through.
func resolve(attr types.String, prefix, suffix string) string {
	if !attr.IsNull() {
		return attr.ValueString()
	}
	if prefix != "" {
		if v := os.Getenv(prefix + "_" + suffix); v != "" {
			return v
		}
	}
	return os.Getenv(envPrefix + "_" + suffix)
}

// envVarName returns the variable name to name in a diagnostic, preferring the
// prefixed one when this provider instance uses a prefix.
func envVarName(prefix, suffix string) string {
	if prefix != "" {
		return prefix + "_" + suffix
	}
	return envPrefix + "_" + suffix
}
