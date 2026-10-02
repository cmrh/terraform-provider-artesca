package account

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AccountDataSource{}

type AccountDataSource struct {
	client *client.ManagementClient
	iam    *client.IAMClient
}

func NewAccountDataSource() datasource.DataSource {
	return &AccountDataSource{}
}

func (d *AccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *AccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing ARTESCA account by name. Email and access keys are not returned -- the account listing does not expose them.",
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
				Description: "The root ARN of the account (arn:aws:iam::<id>:root).",
				Computed:    true,
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
	d.iam = providerData.IAM
}

func (d *AccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := d.client.TokenSource.Token(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading account", fmt.Sprintf("getting auth token: %s", err))
		return
	}

	acct, err := d.iam.GetAccountByName(ctx, token, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading account", err.Error())
		return
	}
	if acct == nil {
		resp.Diagnostics.AddError(
			"Account not found",
			fmt.Sprintf("No account exists with name %q.", data.Name.ValueString()),
		)
		return
	}

	data.Name = types.StringValue(acct.Name)
	data.ID = types.StringValue(acct.ID)
	data.CanonicalID = types.StringValue(acct.CanonicalID)
	data.ARN = types.StringValue(client.AccountARN(acct.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
