package locations

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &LocationsDataSource{}

type LocationsDataSource struct {
	client *client.ManagementClient
}

func NewLocationsDataSource() datasource.DataSource {
	return &LocationsDataSource{}
}

func (d *LocationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ARTESCA storage locations. Returns summary metadata only -- use the singular data.artesca_location to fetch full backend-specific details for a given location.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				Description: "All locations on the management overlay.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":                schema.StringAttribute{Description: "Location name.", Computed: true},
						"location_type":       schema.StringAttribute{Description: "Backend type (e.g. location-aws-s3-v1).", Computed: true},
						"is_builtin":          schema.BoolAttribute{Description: "True if this is a built-in default location.", Computed: true},
						"is_transient":        schema.BoolAttribute{Description: "True if this is a transient location.", Computed: true},
						"legacy_aws_behavior": schema.BoolAttribute{Description: "True if legacy AWS behavior is enabled.", Computed: true},
						"size_limit_gb":       schema.Int64Attribute{Description: "Storage size limit in gigabytes (0 = unlimited).", Computed: true},
						"object_id":           schema.StringAttribute{Description: "Internal object identifier.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *LocationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LocationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	overlay, err := d.client.GetOverlay(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading locations", err.Error())
		return
	}

	out := LocationsDataSourceModel{
		Locations: make([]LocationSummary, 0, len(overlay.Locations)),
	}
	for name, loc := range overlay.Locations {
		// overlay.Locations is keyed by name; the Location struct's Name may
		// not be populated by the API, so prefer the map key when empty.
		locName := loc.Name
		if locName == "" {
			locName = name
		}
		out.Locations = append(out.Locations, LocationSummary{
			Name:              types.StringValue(locName),
			LocationType:      types.StringValue(loc.LocationType),
			IsBuiltin:         types.BoolValue(loc.IsBuiltin),
			IsTransient:       types.BoolValue(loc.IsTransient),
			LegacyAwsBehavior: types.BoolValue(loc.LegacyAwsBehavior),
			SizeLimitGB:       types.Int64Value(loc.SizeLimitGB),
			ObjectID:          types.StringValue(loc.ObjectID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
