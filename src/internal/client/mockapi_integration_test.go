package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

// These tests drive the real client against the Python mock in mockapi/, which
// proves the two agree on the wire: that the shapes the mock returns actually
// deserialize into the client's structs, and that the response shapes which
// differ from the request shapes are handled.
//
// Skipped unless TENABLEIO_MOCK_URL is set, so `make test` stays credential-
// and network-free:
//
//	cd mockapi && make run                     # in one shell
//	TENABLEIO_MOCK_URL=http://127.0.0.1:8080 go test ./internal/client/ -run Mock -v
//
// Each test resets the mock first, so they are order-independent.
func mockClient(t *testing.T) *client.Client {
	t.Helper()
	base := os.Getenv("TENABLEIO_MOCK_URL")
	if base == "" {
		t.Skip("set TENABLEIO_MOCK_URL to run the mock API integration tests")
	}
	resetMock(t, base)
	return client.New("access", "secret", base, "", "", "test")
}

func resetMock(t *testing.T, base string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/__mock/reset", nil)
	if err != nil {
		t.Fatalf("building reset request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("resetting mock at %s: %v (is it running?)", base, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resetting mock: status %d", resp.StatusCode)
	}
}

func TestMockTagCategoryRoundTrip(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	created, err := c.CreateTagCategory(ctx, client.TagCategoryCreateRequest{
		Name:        "Location",
		Description: "Where the asset lives",
	})
	if err != nil {
		t.Fatalf("creating category: %v", err)
	}
	if created.Name != "Location" {
		t.Errorf("name = %q, want %q (the API must echo it verbatim)", created.Name, "Location")
	}
	if created.Description != "Where the asset lives" {
		t.Errorf("description = %q", created.Description)
	}
	if created.UUID == "" || created.CreatedAt == "" {
		t.Errorf("computed fields not populated: %+v", created)
	}

	fetched, err := c.GetTagCategory(ctx, created.UUID)
	if err != nil {
		t.Fatalf("getting category: %v", err)
	}
	if *fetched != *created {
		t.Errorf("get returned %+v, want %+v", fetched, created)
	}
}

func TestMockDuplicateCategoryIsRejected(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	if _, err := c.CreateTagCategory(ctx, client.TagCategoryCreateRequest{Name: "Location"}); err != nil {
		t.Fatalf("creating category: %v", err)
	}
	// Tenable.io answers 400 here; it does not return the existing category.
	if _, err := c.CreateTagCategory(ctx, client.TagCategoryCreateRequest{Name: "Location"}); err == nil {
		t.Fatal("expected an error creating a duplicate category name")
	}
}

// TestMockDynamicTagFilterRoundTrip is the important one: the request carries a
// filters object, the response carries a JSON-formatted string using "field"
// keys and short operator codes, and ParseAssetRules has to bridge the two.
func TestMockDynamicTagFilterRoundTrip(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	category, err := c.CreateTagCategory(ctx, client.TagCategoryCreateRequest{Name: "Location"})
	if err != nil {
		t.Fatalf("creating category: %v", err)
	}

	created, err := c.CreateTagValue(ctx, client.TagValueCreateRequest{
		CategoryUUID: category.UUID,
		Value:        "London",
		Filters: &client.TagValueFilters{Asset: client.TagAssetRules{
			And: []client.TagRule{
				{Property: "operating_system", Operator: "equals", Values: []string{"FreeBSD"}},
				{Property: "ipv4", Operator: "eq", Values: []string{"10.0.0.1", "10.0.0.2"}},
			},
		}},
	})
	if err != nil {
		t.Fatalf("creating tag value: %v", err)
	}
	if created.Type != "dynamic" {
		t.Errorf("type = %q, want dynamic (filters make a tag dynamic)", created.Type)
	}

	fetched, err := c.GetTagValue(ctx, created.UUID)
	if err != nil {
		t.Fatalf("getting tag value: %v", err)
	}

	// The echo is a string, not an object.
	if fetched.Filters == nil || fetched.Filters.Asset == "" {
		t.Fatalf("filters not echoed: %+v", fetched.Filters)
	}
	if !json.Valid([]byte(fetched.Filters.Asset)) {
		t.Fatalf("filters.asset is not valid JSON: %q", fetched.Filters.Asset)
	}

	rules, err := fetched.Filters.ParseAssetRules()
	if err != nil {
		t.Fatalf("parsing asset rules: %v", err)
	}
	if len(rules.And) != 2 {
		t.Fatalf("got %d and-rules, want 2", len(rules.And))
	}

	// "field" in the response must land in Property, and a single value that the
	// API collapsed to a bare string must come back as a one-element slice.
	first := rules.And[0]
	if first.Property != "operating_system" {
		t.Errorf("property = %q, want operating_system (parsed from \"field\")", first.Property)
	}
	if first.Operator != "eq" {
		t.Errorf("operator = %q, want the short code eq", first.Operator)
	}
	if len(first.Values) != 1 || first.Values[0] != "FreeBSD" {
		t.Errorf("values = %v, want [FreeBSD] (collapsed bare string)", first.Values)
	}

	if second := rules.And[1]; len(second.Values) != 2 {
		t.Errorf("values = %v, want two elements preserved as an array", second.Values)
	}
}

// TestMockScanShapesDiffer pins the key renames between POST and GET.
func TestMockScanShapesDiffer(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	folder, err := c.CreateFolder(ctx, "Terraform")
	if err != nil {
		t.Fatalf("creating folder: %v", err)
	}

	created, err := c.CreateScan(ctx, client.ScanCreateRequest{
		UUID: "template-uuid",
		Settings: client.ScanSettings{
			Name:        "Nightly",
			FolderID:    folder.ID,
			TextTargets: "10.0.0.0/24",
			Emails:      "ops@example.com",
		},
	})
	if err != nil {
		t.Fatalf("creating scan: %v", err)
	}
	if created.Scan.TextTargets != "10.0.0.0/24" {
		t.Errorf("create echoed text_targets = %q", created.Scan.TextTargets)
	}

	details, err := c.GetScan(ctx, created.Scan.ID)
	if err != nil {
		t.Fatalf("getting scan: %v", err)
	}
	// GET renames: id->object_id, text_targets->targets,
	// emails->notification_email_address, template->scanner_name.
	if details.Info.ID != created.Scan.ID {
		t.Errorf("object_id = %d, want %d", details.Info.ID, created.Scan.ID)
	}
	if details.Info.Targets != "10.0.0.0/24" {
		t.Errorf("targets = %q", details.Info.Targets)
	}
	if details.Info.Emails != "ops@example.com" {
		t.Errorf("notification_email_address = %q", details.Info.Emails)
	}
	if details.Info.TemplateUUID != "template-uuid" {
		t.Errorf("scanner_name = %q", details.Info.TemplateUUID)
	}
}

func TestMockAssetTagFiltersCatalogue(t *testing.T) {
	c := mockClient(t)

	resp, err := c.ListAssetTagFilters(context.Background())
	if err != nil {
		t.Fatalf("listing asset tag filters: %v", err)
	}
	if len(resp.Filters) == 0 {
		t.Fatal("no filters returned")
	}

	var sawDropdown, sawEntry bool
	for _, f := range resp.Filters {
		if f.Name == "" || len(f.Operators) == 0 {
			t.Errorf("filter %+v is missing a name or operators", f)
		}
		switch f.Control.Type {
		case "dropdown", "dropdown_multi":
			sawDropdown = true
			for _, entry := range f.Control.List {
				// Entries arrive as {name, value} objects; the custom
				// unmarshaller also normalises bare strings.
				if entry.Name == "" || entry.Value == "" {
					t.Errorf("dropdown entry not normalised: %+v", entry)
				}
			}
		case "entry":
			sawEntry = true
		}
	}
	if !sawDropdown || !sawEntry {
		t.Error("expected both dropdown and entry control types in the catalogue")
	}
}

func TestMockSeededReadOnlyEndpoints(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	scanners, err := c.ListScanners(ctx)
	if err != nil {
		t.Fatalf("listing scanners: %v", err)
	}
	if len(scanners.Scanners) == 0 {
		t.Error("expected seeded scanners")
	}

	assets, err := c.ListAssets(ctx, 0)
	if err != nil {
		t.Fatalf("listing assets: %v", err)
	}
	if len(assets.Assets) == 0 {
		t.Error("expected seeded assets")
	}

	networks, err := c.ListNetworks(ctx)
	if err != nil {
		t.Fatalf("listing networks: %v", err)
	}
	if len(networks.Networks) != 1 || !networks.Networks[0].IsDefault {
		t.Errorf("expected exactly one seeded default network, got %+v", networks.Networks)
	}
}
