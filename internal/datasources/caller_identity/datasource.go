package caller_identity

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &CallerIdentityDataSource{}

type CallerIdentityDataSource struct {
	client *client.STSClient
}

func NewCallerIdentityDataSource() datasource.DataSource {
	return &CallerIdentityDataSource{}
}

func (d *CallerIdentityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_caller_identity"
}

func (d *CallerIdentityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the identity associated with the supplied access key (account ID, user ID, and ARN) via STS GetCallerIdentity.",
		Attributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Description: "The access key whose identity should be looked up.",
				Required:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "The secret key paired with access_key.",
				Required:    true,
				Sensitive:   true,
			},
			"session_token": schema.StringAttribute{
				Description: "Optional session token. Set this when introspecting temporary credentials minted by sts:AssumeRole (e.g. those returned by ephemeral.artesca_assumed_role_credentials).",
				Optional:    true,
				Sensitive:   true,
			},
			"user_id": schema.StringAttribute{
				Description: "The unique identifier of the calling principal.",
				Computed:    true,
			},
			"account": schema.StringAttribute{
				Description: "The account ID the calling principal belongs to.",
				Computed:    true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the calling principal.",
				Computed:    true,
			},
		},
	}
}

func (d *CallerIdentityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = providerData.STS
}

func (d *CallerIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CallerIdentityDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identity, err := d.client.GetCallerIdentity(ctx, data.AccessKey.ValueString(), data.SecretKey.ValueString(), data.SessionToken.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error calling STS GetCallerIdentity", err.Error())
		return
	}

	data.UserID = types.StringValue(identity.UserID)
	data.Account = types.StringValue(identity.Account)
	data.ARN = types.StringValue(identity.Arn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
