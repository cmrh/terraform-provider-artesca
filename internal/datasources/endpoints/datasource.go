package endpoints

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
)

var _ datasource.DataSource = &EndpointsDataSource{}

type EndpointsDataSource struct {
	client *client.ManagementClient
}

func NewEndpointsDataSource() datasource.DataSource {
	return &EndpointsDataSource{}
}

func (d *EndpointsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoints"
}

func (d *EndpointsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ARTESCA bucket endpoints (per-bucket DNS hostnames). Includes built-in endpoints (`s3.<cluster>`) alongside user-created ones.",
		Attributes: map[string]schema.Attribute{
			"endpoints": schema.ListNestedAttribute{
				Description: "All endpoints on the management overlay.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"hostname":      schema.StringAttribute{Description: "Endpoint hostname.", Computed: true},
						"location_name": schema.StringAttribute{Description: "Storage location backing the endpoint.", Computed: true},
						"is_builtin":    schema.BoolAttribute{Description: "True if this is a built-in cluster endpoint.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *EndpointsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
	d.client = providerData.Management
}

func (d *EndpointsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	overlay, err := d.client.GetOverlay(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading endpoints", err.Error())
		return
	}

	out := EndpointsDataSourceModel{
		Endpoints: make([]EndpointSummary, 0, len(overlay.Endpoints)),
	}
	for _, ep := range overlay.Endpoints {
		out.Endpoints = append(out.Endpoints, EndpointSummary{
			Hostname:     types.StringValue(ep.Hostname),
			LocationName: types.StringValue(ep.LocationName),
			IsBuiltin:    types.BoolValue(ep.IsBuiltin),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
