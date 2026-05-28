package instance

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &InstanceDataSource{}

type InstanceDataSource struct {
	client *client.ManagementClient
}

func NewInstanceDataSource() datasource.DataSource {
	return &InstanceDataSource{}
}

func (d *InstanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *InstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns metadata and status for an ARTESCA instance. Combines GET /instance/{uuid} (identity + state) and GET /instance/{uuid}/status (live snapshot).",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				Description: "Instance ID. Defaults to the provider's instance_id if omitted.",
				Optional:    true,
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Instance creation timestamp (RFC 3339).",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Provisioning state of the instance (e.g. \"confirmed\", \"provisioning\", \"new\").",
				Computed:    true,
			},
			"public_key": schema.StringAttribute{
				Description: "PEM-encoded public key used to verify instance-issued tokens.",
				Computed:    true,
			},
			"ip_address": schema.StringAttribute{
				Description: "Most recently reported IP address.",
				Computed:    true,
			},
			"last_seen": schema.StringAttribute{
				Description: "Timestamp of the most recent status report (RFC 3339).",
				Computed:    true,
			},
			"running_configuration_version": schema.Int64Attribute{
				Description: "Currently running configuration overlay version.",
				Computed:    true,
			},
			"server_version": schema.StringAttribute{
				Description: "Server build identifier. May be a release version on production builds or a git ref on development builds.",
				Computed:    true,
			},
		},
	}
}

func (d *InstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()
	if instanceID == "" {
		instanceID = d.client.InstanceID
	}

	inst, err := d.client.GetInstance(ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance", err.Error())
		return
	}

	st, err := d.client.GetInstanceStatus(ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance status", err.Error())
		return
	}

	data.InstanceID = types.StringValue(inst.InstanceID)
	data.CreatedAt = types.StringValue(inst.CreatedAt)
	data.State = types.StringValue(inst.State)
	data.PublicKey = types.StringValue(inst.PublicKey)
	data.IPAddress = types.StringValue(st.IPAddress)
	data.LastSeen = types.StringValue(st.LastSeen)
	data.RunningConfigurationVersion = types.Int64Value(st.RunningConfigurationVersion)
	data.ServerVersion = types.StringValue(st.ServerVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
