package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/tenable/terraform-provider-tenableio/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tenableio": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	server, err := testAccProtoV6ProviderFactories["tenableio"]()
	if err != nil {
		t.Fatalf("creating provider server: %s", err)
	}

	resp, err := server.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("getting provider schema: %s", err)
	}

	if resp.Provider == nil {
		t.Fatal("provider schema is nil")
	}

	providerAttrs := map[string]bool{"access_key": false, "secret_key": false, "base_url": false}
	for _, attr := range resp.Provider.Block.Attributes {
		if _, ok := providerAttrs[attr.Name]; ok {
			providerAttrs[attr.Name] = true
		}
	}
	for name, found := range providerAttrs {
		if !found {
			t.Errorf("expected provider attribute %q not found", name)
		}
	}

	expectedResources := []string{
		"tenableio_scan", "tenableio_policy", "tenableio_folder",
		"tenableio_exclusion", "tenableio_network",
		"tenableio_tag_category", "tenableio_tag_value",
		"tenableio_agent_group",
	}
	for _, name := range expectedResources {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("expected resource %q not registered", name)
		}
	}

	expectedDataSources := []string{
		"tenableio_scans", "tenableio_policies", "tenableio_asset", "tenableio_assets",
		"tenableio_folders", "tenableio_exclusions", "tenableio_networks",
		"tenableio_scanners", "tenableio_agent_groups",
		"tenableio_tag_categories", "tenableio_tag_values",
	}
	for _, name := range expectedDataSources {
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("expected data source %q not registered", name)
		}
	}
}
