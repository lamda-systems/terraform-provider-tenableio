package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
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

	const description = "daily scan on Tenable test devices"
	const scanTimeWindow = 180

	created, err := c.CreateScan(ctx, client.ScanCreateRequest{
		UUID: "template-uuid",
		Settings: client.ScanSettings{
			Name:           "Nightly",
			Description:    description,
			FolderID:       folder.ID,
			TextTargets:    "10.0.0.0/24",
			Emails:         "ops@example.com",
			ScanTimeWindow: scanTimeWindow,
		},
	})
	if err != nil {
		t.Fatalf("creating scan: %v", err)
	}
	if created.Scan.TextTargets != "10.0.0.0/24" {
		t.Errorf("create echoed text_targets = %q", created.Scan.TextTargets)
	}
	if created.Scan.Description == nil || *created.Scan.Description != description {
		t.Errorf("create echoed description = %s, want %q", quoteOrNil(created.Scan.Description), description)
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
	assertReported(t, "targets", details.Info.Targets, "10.0.0.0/24")
	assertReported(t, "notification_email_address", details.Info.Emails, "ops@example.com")
	assertReported(t, "scanner_name", details.Info.TemplateUUID, "template-uuid")

	// The details endpoint reports neither the description nor the submitted
	// settings. Whether a field is absent (live behaviour, and the mock's
	// default) or present because MOCK_SCAN_DETAILS_SETTINGS=report, it must
	// never come back as a zero value for a scan that has one: that reading
	// made the provider report a described scan as wiped and re-propose a
	// configured scan_time_window on every plan.
	assertReported(t, "details description", details.Info.Description, description)
	if got := details.Info.ScanTimeWindow; got != nil && *got != scanTimeWindow {
		t.Errorf("details reported scan_time_window = %d, want %d or nothing at all", *got, scanTimeWindow)
	}
	if got := details.Info.CreationDate; got != nil && *got == 0 {
		t.Error("details reported creation_date = 0; absent would be honest, zero is not")
	}

	// The update echo is optional -- several tenants answer with no body -- but
	// when it is there it is the only place a description comes back from.
	updated, err := c.UpdateScan(ctx, created.Scan.ID, client.ScanUpdateRequest{
		Settings: client.ScanSettings{
			Name:        "Nightly",
			Description: description,
			FolderID:    folder.ID,
			TextTargets: "10.0.0.0/24",
		},
	})
	if err != nil {
		t.Fatalf("updating scan: %v", err)
	}
	if updated.Scan != nil {
		if got := updated.Scan.Description; got != nil && *got != description {
			t.Errorf("update echoed description = %q, want %q", *got, description)
		}
	}
}

func quoteOrNil(p *string) string {
	if p == nil {
		return "nothing"
	}
	return fmt.Sprintf("%q", *p)
}

// assertReported accepts a field the endpoint may legitimately omit, and rejects
// the one answer that is never right: reported, but empty, for a value that was
// set. That is the shape which read back as "the tenant wiped it".
func assertReported(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got != nil && *got != want {
		t.Errorf("%s = %q, want %q or nothing at all", label, *got, want)
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

func TestMockTemplateLists(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	byName := map[client.TemplateType]map[string]client.Template{}
	for _, templateType := range []client.TemplateType{client.TemplateTypeScan, client.TemplateTypePolicy} {
		resp, err := c.ListTemplates(ctx, templateType)
		if err != nil {
			t.Fatalf("listing %s templates: %v", templateType, err)
		}
		byName[templateType] = map[string]client.Template{}
		for _, tmpl := range resp.Templates {
			if tmpl.UUID == "" || tmpl.Name == "" || tmpl.Title == "" {
				t.Errorf("%s template %+v is missing a uuid, name or title", templateType, tmpl)
			}
			byName[templateType][tmpl.Name] = tmpl
		}
	}

	scan, policy := byName[client.TemplateTypeScan]["advanced"], byName[client.TemplateTypePolicy]["advanced"]
	if scan.UUID == "" || policy.UUID == "" {
		t.Fatal("expected the advanced template in both lists")
	}
	if scan.UUID == policy.UUID {
		t.Error("the mock gives each list its own uuids; a shared one means the lists were mixed up")
	}

	// null on the wire must survive as nil, not collapse to false.
	if basic := byName[client.TemplateTypeScan]["basic"]; basic.IsAgent != nil {
		t.Errorf("basic.IsAgent = %v, want nil", *basic.IsAgent)
	}
	if agent := byName[client.TemplateTypeScan]["agent_basic"]; agent.IsAgent == nil || !*agent.IsAgent {
		t.Errorf("agent_basic.IsAgent = %v, want true", agent.IsAgent)
	}
	if agent := byName[client.TemplateTypeScan]["agent_basic"]; agent.Order != nil {
		t.Errorf("agent_basic.Order = %d, want nil", *agent.Order)
	}
}

func TestMockPolicyListShape(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	base := policyTemplateUUID(t, c, "advanced")
	created, err := c.CreatePolicy(ctx, client.PolicyRequest{
		TemplateUUID: base, Name: "Internal - Credentialed", Description: "weekly", Visibility: "shared",
	})
	if err != nil {
		t.Fatalf("creating policy: %v", err)
	}

	list, err := c.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("listing policies: %v", err)
	}
	var found *client.PolicySummary
	for i := range list.Policies {
		if list.Policies[i].ID == created.PolicyID {
			found = &list.Policies[i]
		}
	}
	if found == nil {
		t.Fatalf("policy %d missing from the list", created.PolicyID)
	}
	if found.Name != "Internal - Credentialed" || found.TemplateUUID != base {
		t.Errorf("list entry = %+v, want the name and base template it was created with", *found)
	}
	if found.Description == nil || *found.Description != "weekly" {
		t.Errorf("Description = %v, want \"weekly\"", found.Description)
	}
	if found.Shared != 1 {
		t.Errorf("Shared = %d, want 1 for a shared policy", found.Shared)
	}
}

// policyTemplateUUID looks a template up in the policy list, which is the only
// list a policy may be based on.
func policyTemplateUUID(t *testing.T, c *client.Client, name string) string {
	t.Helper()
	list, err := c.ListTemplates(context.Background(), client.TemplateTypePolicy)
	if err != nil {
		t.Fatalf("listing policy templates: %v", err)
	}
	for _, tmpl := range list.Templates {
		if tmpl.Name == name {
			return tmpl.UUID
		}
	}
	t.Fatalf("no %q template in the policy list", name)
	return ""
}

func TestMockPolicyTemplateRoundTrip(t *testing.T) {
	c := mockClient(t)
	ctx := context.Background()

	created, err := c.CreatePolicy(ctx, client.PolicyRequest{
		TemplateUUID: policyTemplateUUID(t, c, "advanced"),
		Name:         "Internal",
		Settings: map[string]any{
			"ping_the_remote_host":         "no",
			"max_compliance_check_timeout": "600",
			"webcrawl_max_depth":           "4",
		},
		Plugins: map[string]client.PluginFamily{"Backdoors": {Status: "disabled"}},
	})
	if err != nil {
		t.Fatalf("creating policy: %v", err)
	}

	details, err := c.GetPolicyDetails(ctx, created.PolicyID)
	if err != nil {
		t.Fatalf("getting policy details: %v", err)
	}
	for key, want := range map[string]string{
		"name": "Internal", "description": "", "ping_the_remote_host": "no",
		// Sent as a string, reported as an integer, normalised back to a string.
		"max_compliance_check_timeout": "600",
	} {
		if got := details.Setting(key); got == nil || *got != want {
			t.Errorf("setting %s = %v, want %q", key, got, want)
		}
	}
	// Stored, but not reported while scan_webapps is off.
	if got := details.Setting("webcrawl_max_depth"); got != nil {
		t.Errorf("webcrawl_max_depth = %q, want it unreported while its toggle is off", *got)
	}
	if details.Plugins["Backdoors"].Status != "disabled" || details.Plugins["SCADA"].Status != "enabled" {
		t.Errorf("plugins = %v, want Backdoors disabled and the rest at the template's status", details.Plugins)
	}
	if !reflect.DeepEqual(details.PresentKeys, []string{"plugins", "settings", "uuid"}) {
		t.Errorf("PresentKeys = %v, want the documented nesting", details.PresentKeys)
	}

	if err := c.UpdatePolicy(ctx, created.PolicyID, client.PolicyRequest{
		TemplateUUID: "ignored", Name: "Internal", Settings: map[string]any{"scan_webapps": "yes"},
	}); err != nil {
		t.Fatalf("updating policy: %v", err)
	}
	details, err = c.GetPolicyDetails(ctx, created.PolicyID)
	if err != nil {
		t.Fatalf("getting policy details: %v", err)
	}
	if got := details.Setting("ping_the_remote_host"); got == nil || *got != "no" {
		t.Errorf("ping_the_remote_host = %v after an update that left it out, want \"no\" kept", got)
	}
	if got := details.Setting("webcrawl_max_depth"); got == nil || *got != "4" {
		t.Errorf("webcrawl_max_depth = %v once scan_webapps is on, want \"4\"", got)
	}
}
