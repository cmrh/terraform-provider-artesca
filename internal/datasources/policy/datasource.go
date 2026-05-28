package policy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-artesca/internal/client"
)

var _ datasource.DataSource = &PolicyDataSource{}

type PolicyDataSource struct {
	client *client.IAMClient
}

func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

func (d *PolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (d *PolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing IAM managed policy within an ARTESCA account by ARN.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account this policy belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account this policy belongs to.",
				Required:    true,
				Sensitive:   true,
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the IAM managed policy to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the policy.",
				Computed:    true,
			},
			"policy_id": schema.StringAttribute{
				Description: "The unique ID of the policy.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the policy.",
				Computed:    true,
			},
			"default_version_id": schema.StringAttribute{
				Description: "The default version ID of the policy.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the policy.",
				Computed:    true,
			},
			"policy_document": schema.StringAttribute{
				Description: "The JSON policy document for the default version.",
				Computed:    true,
			},
		},
	}
}

func (d *PolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessKey := data.AccountAccessKey.ValueString()
	secretKey := data.AccountSecretKey.ValueString()
	arn := data.ARN.ValueString()

	policy, err := d.client.GetPolicy(ctx, accessKey, secretKey, arn)
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM policy", err.Error())
		return
	}
	if policy == nil {
		resp.Diagnostics.AddError(
			"IAM policy not found",
			fmt.Sprintf("No IAM managed policy exists with ARN %q in this account.", arn),
		)
		return
	}

	document, err := d.client.GetPolicyDocument(ctx, accessKey, secretKey, arn, policy.DefaultVersionId)
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM policy document", err.Error())
		return
	}

	data.Name = types.StringValue(policy.PolicyName)
	data.PolicyID = types.StringValue(policy.PolicyId)
	data.Path = types.StringValue(policy.Path)
	data.DefaultVersionID = types.StringValue(policy.DefaultVersionId)
	data.Description = types.StringValue(policy.Description)
	data.PolicyDocument = types.StringValue(document)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
