package group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &GroupDataSource{}

type GroupDataSource struct {
	client *client.IAMClient
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing IAM group within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account this group belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account this group belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the IAM group to look up.",
				Required:    true,
			},
			"group_id": schema.StringAttribute{
				Description: "The unique ID of the group.",
				Computed:    true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the group.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the group.",
				Computed:    true,
			},
		},
	}
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = providerData.IAM
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := d.client.GetGroup(ctx, data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM group", err.Error())
		return
	}
	if group == nil {
		resp.Diagnostics.AddError(
			"IAM group not found",
			fmt.Sprintf("No IAM group exists with name %q in this account.", data.Name.ValueString()),
		)
		return
	}

	data.GroupID = types.StringValue(group.GroupId)
	data.ARN = types.StringValue(group.Arn)
	data.Path = types.StringValue(group.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
