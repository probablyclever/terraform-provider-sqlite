package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderMetadataSchemaConfigure(t *testing.T) {
	providerImpl := New()
	p, ok := providerImpl.(*sqliteProvider)
	if !ok {
		t.Fatalf("unexpected provider type %T", providerImpl)
	}

	var meta provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &meta)
	if meta.TypeName != "sqlite" {
		t.Fatalf("unexpected provider type %q", meta.TypeName)
	}

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)
	if len(schemaResp.Schema.Attributes) != 0 {
		t.Fatalf("expected empty provider schema")
	}

	p.Configure(context.Background(), provider.ConfigureRequest{}, &provider.ConfigureResponse{})

	if ds := p.DataSources(context.Background()); len(ds) != 1 {
		t.Fatalf("expected one data source, got %d", len(ds))
	}
	if res := p.Resources(context.Background()); res != nil {
		t.Fatalf("expected no resources")
	}
}
