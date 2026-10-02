package user

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct {
	accounts *client.AccountCredentialSource
	client   *client.IAMClient
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing IAM user within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.DataSourceAttribute(),
			"username": schema.StringAttribute{
				Description: "The name of the IAM user to look up.",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The unique ID of the user.",
				Computed:    true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the user.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the user.",
				Computed:    true,
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.accounts = providerData.Accounts
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := d.accounts.For(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	user, err := d.client.GetUser(ctx, acctCreds, data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError(
			"IAM user not found",
			fmt.Sprintf("No IAM user exists with name %q in this account.", data.Username.ValueString()),
		)
		return
	}

	data.UserID = types.StringValue(user.UserId)
	data.ARN = types.StringValue(user.Arn)
	data.Path = types.StringValue(user.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
