package accounts

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AccountsDataSource{}

type AccountsDataSource struct {
	client *client.ManagementClient
	iam    *client.IAMClient
}

func NewAccountsDataSource() datasource.DataSource {
	return &AccountsDataSource{}
}

func (d *AccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accounts"
}

func (d *AccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ARTESCA accounts on the cluster. Email and access keys are not returned -- the account listing does not expose them.",
		Attributes: map[string]schema.Attribute{
			"accounts": schema.ListNestedAttribute{
				Description: "All accounts on the cluster.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         schema.StringAttribute{Description: "Account name.", Computed: true},
						"id":           schema.StringAttribute{Description: "Unique account ID.", Computed: true},
						"canonical_id": schema.StringAttribute{Description: "Canonical ID.", Computed: true},
						"arn":          schema.StringAttribute{Description: "Account root ARN (arn:aws:iam::<id>:root).", Computed: true},
					},
				},
			},
		},
	}
}

func (d *AccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AccountsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	token, err := d.client.TokenSource.Token(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading accounts", fmt.Sprintf("getting auth token: %s", err))
		return
	}

	accounts, err := d.iam.ListAccounts(ctx, token)
	if err != nil {
		resp.Diagnostics.AddError("Error reading accounts", err.Error())
		return
	}

	out := AccountsDataSourceModel{
		Accounts: make([]AccountSummary, 0, len(accounts)),
	}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, AccountSummary{
			Name:        types.StringValue(a.Name),
			ID:          types.StringValue(a.ID),
			CanonicalID: types.StringValue(a.CanonicalID),
			ARN:         types.StringValue(client.AccountARN(a.ID)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
