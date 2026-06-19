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

	attrs := resp.Provider.Block.Attributes
	expected := map[string]bool{"access_key": false, "secret_key": false, "base_url": false}
	for _, attr := range attrs {
		if _, ok := expected[attr.Name]; ok {
			expected[attr.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected attribute %q not found in provider schema", name)
		}
	}
}
