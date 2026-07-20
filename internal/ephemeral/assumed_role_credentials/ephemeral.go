package assumedrolecredentials

import (
	"context"
	"fmt"
	"time"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ ephemeral.EphemeralResource = &AssumedRoleCredentialsEphemeralResource{}

type AssumedRoleCredentialsEphemeralResource struct {
	stsClient *client.STSClient
}

func NewAssumedRoleCredentialsEphemeralResource() ephemeral.EphemeralResource {
	return &AssumedRoleCredentialsEphemeralResource{}
}

func (r *AssumedRoleCredentialsEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assumed_role_credentials"
}

func (r *AssumedRoleCredentialsEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Calls sts:AssumeRole and returns temporary credentials. Values are never written to Terraform state -- they exist only for the duration of the current operation. Requires Terraform/OpenTofu 1.10+.",
		Attributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Description: "Access key of the caller (the IAM user assuming the role).",
				Required:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "Secret key of the caller.",
				Required:    true,
				Sensitive:   true,
			},
			"role_arn": schema.StringAttribute{
				Description: "ARN of the role to assume.",
				Required:    true,
			},
			"role_session_name": schema.StringAttribute{
				Description: "Session name (recorded in the assumed-role ARN; useful for audit).",
				Required:    true,
			},
			"duration_seconds": schema.Int64Attribute{
				Description: "How long the returned credentials are valid. STS defaults to 3600 (1 hour) when omitted.",
				Optional:    true,
			},
			"external_id": schema.StringAttribute{
				Description: "Optional external ID required by the role's trust policy.",
				Optional:    true,
			},
			"policy": schema.StringAttribute{
				Description: "Optional inline session policy (JSON). Further restricts the assumed permissions.",
				Optional:    true,
			},
			"access_key_id": schema.StringAttribute{
				Description: "Returned temporary access key ID.",
				Computed:    true,
				Sensitive:   true,
			},
			"secret_access_key": schema.StringAttribute{
				Description: "Returned temporary secret access key.",
				Computed:    true,
				Sensitive:   true,
			},
			"session_token": schema.StringAttribute{
				Description: "Returned session token. Must be sent alongside the access key / secret on signed requests.",
				Computed:    true,
				Sensitive:   true,
			},
			"expiration": schema.StringAttribute{
				Description: "When the returned credentials expire (RFC3339).",
				Computed:    true,
			},
			"assumed_role_id": schema.StringAttribute{
				Description: "Unique identifier of the assumed role session (RoleId:SessionName).",
				Computed:    true,
			},
			"assumed_role_arn": schema.StringAttribute{
				Description: "ARN of the assumed-role session (arn:aws:sts::<id>:assumed-role/<role>/<session>).",
				Computed:    true,
			},
		},
	}
}

func (r *AssumedRoleCredentialsEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
	if providerData.STS == nil {
		resp.Diagnostics.AddError(
			"STS client not configured",
			"The provider's s3_endpoint must be set so the STS endpoint can be derived from it. Set s3_endpoint in the provider block or ARTESCA_S3_ENDPOINT in the environment.",
		)
		return
	}
	r.stsClient = providerData.STS
}

func (r *AssumedRoleCredentialsEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data AssumedRoleCredentialsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Assuming role", map[string]any{
		"role_arn":          data.RoleArn.ValueString(),
		"role_session_name": data.RoleSessionName.ValueString(),
	})

	opts := client.AssumeRoleOptions{
		DurationSeconds: data.DurationSeconds.ValueInt64(),
		ExternalID:      data.ExternalID.ValueString(),
		Policy:          data.Policy.ValueString(),
	}

	creds, err := r.stsClient.AssumeRole(ctx,
		data.AccessKey.ValueString(),
		data.SecretKey.ValueString(),
		data.RoleArn.ValueString(),
		data.RoleSessionName.ValueString(),
		opts,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error assuming role", err.Error())
		return
	}

	data.AccessKeyID = types.StringValue(creds.AccessKeyID)
	data.SecretAccessKey = types.StringValue(creds.SecretAccessKey)
	data.SessionToken = types.StringValue(creds.SessionToken)
	data.Expiration = types.StringValue(creds.Expiration.Format(time.RFC3339))
	data.AssumedRoleID = types.StringValue(creds.AssumedRoleID)
	data.AssumedRoleArn = types.StringValue(creds.AssumedRoleArn)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
