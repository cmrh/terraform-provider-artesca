package accounts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &AccountsDataSource{}

type AccountsDataSource struct {
	client *client.ManagementClient
}

func NewAccountsDataSource() datasource.DataSource {
	return &AccountsDataSource{}
}

func (d *AccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accounts"
}

func (d *AccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ARTESCA accounts on the cluster. Access keys and secrets are not returned -- the overlay view does not include them.",
		Attributes: map[string]schema.Attribute{
			"accounts": schema.ListNestedAttribute{
				Description: "All accounts visible on the management overlay.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         schema.StringAttribute{Description: "Account name.", Computed: true},
						"id":           schema.StringAttribute{Description: "Unique account ID.", Computed: true},
						"canonical_id": schema.StringAttribute{Description: "Canonical ID.", Computed: true},
						"arn":          schema.StringAttribute{Description: "Account ARN as returned by the API.", Computed: true},
						"email":        schema.StringAttribute{Description: "Account email.", Computed: true},
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
}

func (d *AccountsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	overlay, err := d.client.GetOverlay(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading accounts", err.Error())
		return
	}

	out := AccountsDataSourceModel{
		Accounts: make([]AccountSummary, 0, len(overlay.Users)),
	}
	for _, u := range overlay.Users {
		name := u.AccountName
		if name == "" {
			name = u.UserName
		}
		out.Accounts = append(out.Accounts, AccountSummary{
			Name:        types.StringValue(name),
			ID:          types.StringValue(u.ID),
			CanonicalID: types.StringValue(u.CanonicalID),
			ARN:         types.StringValue(u.ARN),
			Email:       types.StringValue(u.Email),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
