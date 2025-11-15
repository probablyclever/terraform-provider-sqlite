package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type sqliteProvider struct{}

func New() provider.Provider { return &sqliteProvider{} }

func (p *sqliteProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sqlite"
}

func (p *sqliteProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *sqliteProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// no-op
	resp.DataSourceData = nil
	resp.ResourceData = nil
}

func (p *sqliteProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSQLiteQueryDataSource,
	}
}

func (p *sqliteProvider) Resources(context.Context) []func() resource.Resource { return nil }
