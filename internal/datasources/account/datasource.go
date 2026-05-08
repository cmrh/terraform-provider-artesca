package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
)

var _ datasource.DataSource = &AccountDataSource{}

type AccountDataSource struct {
	client *client.ManagementClient
}

func NewAccountDataSource() datasource.DataSource {
	return &AccountDataSource{}
}

func (d *AccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *AccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing ARTESCA account by name. The account's secret key is not returned -- the API does not expose it after creation.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the account to look up.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The unique ID of the account.",
				Computed:    true,
			},
			"canonical_id": schema.StringAttribute{
				Description: "The canonical ID of the account.",
				Computed:    true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the account.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email associated with the account.",
				Computed:    true,
			},
			"access_key": schema.StringAttribute{
				Description: "The account's access key.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *AccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetAccount(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading account", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError(
			"Account not found",
			fmt.Sprintf("No account exists with name %q.", data.Name.ValueString()),
		)
		return
	}

	if user.AccountName != "" {
		data.Name = types.StringValue(user.AccountName)
	} else if user.UserName != "" {
		data.Name = types.StringValue(user.UserName)
	}
	data.ID = types.StringValue(user.ID)
	data.CanonicalID = types.StringValue(user.CanonicalID)
	data.ARN = types.StringValue(user.ARN)
	data.Email = types.StringValue(user.Email)
	data.AccessKey = types.StringValue(user.AccessKey)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
