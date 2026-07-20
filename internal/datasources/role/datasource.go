package role

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoleDataSource{}

type RoleDataSource struct {
	client *client.IAMClient
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing IAM role within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account this role belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account this role belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the IAM role to look up.",
				Required:    true,
			},
			"role_id": schema.StringAttribute{
				Description: "The unique ID of the role.",
				Computed:    true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the role.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the role.",
				Computed:    true,
			},
			"assume_role_policy_document": schema.StringAttribute{
				Description: "The trust policy document for the role.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the role.",
				Computed:    true,
			},
		},
	}
}

func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.GetRole(ctx, data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError(
			"IAM role not found",
			fmt.Sprintf("No IAM role exists with name %q in this account.", data.Name.ValueString()),
		)
		return
	}

	data.RoleID = types.StringValue(role.RoleId)
	data.ARN = types.StringValue(role.Arn)
	data.Path = types.StringValue(role.Path)
	data.AssumeRolePolicyDocument = types.StringValue(role.AssumeRolePolicyDocument)
	data.Description = types.StringValue(role.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
