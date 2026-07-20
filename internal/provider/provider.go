package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	accountds "github.com/cmrh/terraform-provider-artesca/internal/datasources/account"
	accountsds "github.com/cmrh/terraform-provider-artesca/internal/datasources/accounts"
	bucketworkflowsds "github.com/cmrh/terraform-provider-artesca/internal/datasources/bucket_workflows"
	calleridentityds "github.com/cmrh/terraform-provider-artesca/internal/datasources/caller_identity"
	endpointsds "github.com/cmrh/terraform-provider-artesca/internal/datasources/endpoints"
	groupds "github.com/cmrh/terraform-provider-artesca/internal/datasources/group"
	instanceds "github.com/cmrh/terraform-provider-artesca/internal/datasources/instance"
	locationds "github.com/cmrh/terraform-provider-artesca/internal/datasources/location"
	locationsds "github.com/cmrh/terraform-provider-artesca/internal/datasources/locations"
	policyds "github.com/cmrh/terraform-provider-artesca/internal/datasources/policy"
	roleds "github.com/cmrh/terraform-provider-artesca/internal/datasources/role"
	userds "github.com/cmrh/terraform-provider-artesca/internal/datasources/user"
	assumedrolecredentials "github.com/cmrh/terraform-provider-artesca/internal/ephemeral/assumed_role_credentials"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/account"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/bucket"
	bucketencryption "github.com/cmrh/terraform-provider-artesca/internal/resources/bucket_encryption"
	bucketpolicy "github.com/cmrh/terraform-provider-artesca/internal/resources/bucket_policy"
	buckettagging "github.com/cmrh/terraform-provider-artesca/internal/resources/bucket_tagging"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/endpoint"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/group"
	groupmembership "github.com/cmrh/terraform-provider-artesca/internal/resources/group_membership"
	grouppolicy "github.com/cmrh/terraform-provider-artesca/internal/resources/group_policy"
	grouppolicyattachment "github.com/cmrh/terraform-provider-artesca/internal/resources/group_policy_attachment"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/location"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/policy"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/replication"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/role"
	rolepolicyattachment "github.com/cmrh/terraform-provider-artesca/internal/resources/role_policy_attachment"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/user"
	useraccesskey "github.com/cmrh/terraform-provider-artesca/internal/resources/user_access_key"
	userpolicy "github.com/cmrh/terraform-provider-artesca/internal/resources/user_policy"
	userpolicyattachment "github.com/cmrh/terraform-provider-artesca/internal/resources/user_policy_attachment"
	workflowexpiration "github.com/cmrh/terraform-provider-artesca/internal/resources/workflow_expiration"
	workflowreplication "github.com/cmrh/terraform-provider-artesca/internal/resources/workflow_replication"
	workflowtransition "github.com/cmrh/terraform-provider-artesca/internal/resources/workflow_transition"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &ArtescaProvider{}

type ArtescaProvider struct {
	version string
}

type ArtescaProviderModel struct {
	ManagementEndpoint types.String `tfsdk:"management_endpoint"`
	InstanceID         types.String `tfsdk:"instance_id"`
	OIDCUrl            types.String `tfsdk:"oidc_url"`
	OIDCRealm          types.String `tfsdk:"oidc_realm"`
	ClientID           types.String `tfsdk:"client_id"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	IAMRegion          types.String `tfsdk:"iam_region"`
	S3Endpoint         types.String `tfsdk:"s3_endpoint"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ArtescaProvider{
			version: version,
		}
	}
}

func (p *ArtescaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "artesca"
	resp.Version = p.version
}

func (p *ArtescaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Scality ARTESCA resources via the management API.",
		Attributes: map[string]schema.Attribute{
			"management_endpoint": schema.StringAttribute{
				Description: "The URL of the ARTESCA management API (e.g., https://management.artesca.example.com). Can also be set with ARTESCA_MANAGEMENT_ENDPOINT.",
				Optional:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: "The ARTESCA instance UUID. If omitted, the provider will auto-discover it. Can also be set with ARTESCA_INSTANCE_ID.",
				Optional:    true,
			},
			"oidc_url": schema.StringAttribute{
				Description: "The base URL of the OIDC provider (e.g., https://ui.artesca.example.com). Can also be set with ARTESCA_OIDC_URL.",
				Optional:    true,
			},
			"oidc_realm": schema.StringAttribute{
				Description: "The OIDC realm name. Defaults to 'artesca'. Can also be set with ARTESCA_OIDC_REALM.",
				Optional:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OIDC client ID. Defaults to 'zenko-ui'. Can also be set with ARTESCA_CLIENT_ID.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The OIDC username. Can also be set with ARTESCA_USERNAME.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The OIDC password. Can also be set with ARTESCA_PASSWORD.",
				Optional:    true,
				Sensitive:   true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Defaults to false. Can also be set with ARTESCA_INSECURE_SKIP_VERIFY.",
				Optional:    true,
			},
			"iam_region": schema.StringAttribute{
				Description: "The AWS region used for IAM SigV4 request signing. Defaults to 'us-east-1'. Can also be set with ARTESCA_IAM_REGION.",
				Optional:    true,
			},
			"s3_endpoint": schema.StringAttribute{
				Description: "The ARTESCA S3 endpoint URL (e.g., https://s3.artesca.example.com). Required for bucket resources. Can also be set with ARTESCA_S3_ENDPOINT.",
				Optional:    true,
			},
		},
	}
}

func (p *ArtescaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ArtescaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementEndpoint := envOrValue(config.ManagementEndpoint, "ARTESCA_MANAGEMENT_ENDPOINT")
	instanceID := envOrValue(config.InstanceID, "ARTESCA_INSTANCE_ID")
	oidcURL := envOrValue(config.OIDCUrl, "ARTESCA_OIDC_URL")
	oidcRealm := envOrDefault(config.OIDCRealm, "ARTESCA_OIDC_REALM", "artesca")
	clientID := envOrDefault(config.ClientID, "ARTESCA_CLIENT_ID", "zenko-ui")
	oidcScope := os.Getenv("ARTESCA_OIDC_SCOPE")
	if oidcScope == "" {
		oidcScope = "openid"
	}
	username := envOrValue(config.Username, "ARTESCA_USERNAME")
	password := envOrValue(config.Password, "ARTESCA_PASSWORD")
	iamRegion := envOrDefault(config.IAMRegion, "ARTESCA_IAM_REGION", "us-east-1")
	s3Endpoint := envOrValue(config.S3Endpoint, "ARTESCA_S3_ENDPOINT")

	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() && !config.InsecureSkipVerify.IsUnknown() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	} else if v := os.Getenv("ARTESCA_INSECURE_SKIP_VERIFY"); v == "true" || v == "1" {
		insecureSkipVerify = true
	}

	if managementEndpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Management Endpoint",
			"The management_endpoint must be set in the provider configuration or via the ARTESCA_MANAGEMENT_ENDPOINT environment variable.",
		)
	}
	if oidcURL == "" {
		resp.Diagnostics.AddError(
			"Missing OIDC URL",
			"The oidc_url must be set in the provider configuration or via the ARTESCA_OIDC_URL environment variable.",
		)
	}
	if username == "" {
		resp.Diagnostics.AddError(
			"Missing Username",
			"The username must be set in the provider configuration or via the ARTESCA_USERNAME environment variable.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddError(
			"Missing Password",
			"The password must be set in the provider configuration or via the ARTESCA_PASSWORD environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configuring ARTESCA provider", map[string]interface{}{
		"management_endpoint":  managementEndpoint,
		"oidc_url":             oidcURL,
		"oidc_realm":           oidcRealm,
		"insecure_skip_verify": insecureSkipVerify,
	})

	tokenSource := client.NewOIDCTokenSource(oidcURL, oidcRealm, clientID, oidcScope, username, password, insecureSkipVerify)

	// Validate credentials by fetching an initial token.
	_, err := tokenSource.Token(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"OIDC Authentication Failed",
			fmt.Sprintf("Failed to obtain initial OIDC token: %s\n\nPlease verify your oidc_url, username, and password.", err),
		)
		return
	}

	mgmtClient := client.NewManagementClient(managementEndpoint, instanceID, tokenSource, insecureSkipVerify)

	// Auto-discover instance ID if not provided, by extracting it from the JWT token.
	if instanceID == "" {
		tflog.Info(ctx, "Instance ID not provided, extracting from OIDC token")
		instanceIDs, err := tokenSource.InstanceIDs(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Instance ID Discovery Failed",
				fmt.Sprintf("Failed to extract instance ID from OIDC token: %s\n\nPlease set instance_id explicitly in the provider configuration.", err),
			)
			return
		}
		if len(instanceIDs) == 0 {
			resp.Diagnostics.AddError(
				"Instance ID Discovery Failed",
				"The OIDC token does not contain any instanceIds. Please set instance_id explicitly in the provider configuration.",
			)
			return
		}
		if len(instanceIDs) > 1 {
			resp.Diagnostics.AddError(
				"Instance ID Discovery Failed",
				fmt.Sprintf("The OIDC token contains multiple instance IDs (%d). Please set instance_id explicitly in the provider configuration.", len(instanceIDs)),
			)
			return
		}
		mgmtClient.InstanceID = instanceIDs[0]
		tflog.Info(ctx, "Discovered instance ID from OIDC token", map[string]any{
			"instance_id": instanceIDs[0],
		})
	}

	// Derive IAM endpoint from management endpoint.
	iamEndpoint, err := client.DeriveIAMEndpoint(managementEndpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"IAM Endpoint Derivation Failed",
			fmt.Sprintf("Failed to derive IAM endpoint from management endpoint: %s", err),
		)
		return
	}
	tflog.Info(ctx, "Derived IAM endpoint", map[string]any{"iam_endpoint": iamEndpoint})

	iamClient := client.NewIAMClient(iamEndpoint, iamRegion, insecureSkipVerify)

	var s3Client *client.S3Client
	var stsClient *client.STSClient
	if s3Endpoint != "" {
		s3Client = client.NewS3Client(s3Endpoint, iamRegion, insecureSkipVerify)
		tflog.Info(ctx, "Configured S3 client", map[string]any{"s3_endpoint": s3Endpoint})

		// STS endpoint is derived from the S3 endpoint (s3.<cluster> -> sts.<cluster>).
		stsEndpoint, err := client.DeriveSTSEndpoint(s3Endpoint)
		if err != nil {
			tflog.Warn(ctx, "STS endpoint derivation failed; STS-backed ephemeral resources will be unavailable", map[string]any{"error": err.Error()})
		} else {
			stsClient = client.NewSTSClient(stsEndpoint, iamRegion, insecureSkipVerify)
			tflog.Info(ctx, "Configured STS client", map[string]any{"sts_endpoint": stsEndpoint})
		}
	}

	clients := &client.ProviderClients{
		Management: mgmtClient,
		IAM:        iamClient,
		S3:         s3Client,
		STS:        stsClient,
	}

	resp.ResourceData = clients
	resp.DataSourceData = clients
	resp.EphemeralResourceData = clients
}

func (p *ArtescaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		account.NewAccountResource,
		bucket.NewBucketResource,
		bucketencryption.NewBucketEncryptionResource,
		bucketpolicy.NewBucketPolicyResource,
		buckettagging.NewBucketTaggingResource,
		endpoint.NewEndpointResource,
		group.NewGroupResource,
		groupmembership.NewGroupMembershipResource,
		grouppolicy.NewGroupPolicyResource,
		grouppolicyattachment.NewGroupPolicyAttachmentResource,
		location.NewLocationResource,
		policy.NewPolicyResource,
		replication.NewReplicationResource,
		role.NewRoleResource,
		rolepolicyattachment.NewRolePolicyAttachmentResource,
		user.NewUserResource,
		useraccesskey.NewUserAccessKeyResource,
		userpolicy.NewUserPolicyResource,
		userpolicyattachment.NewUserPolicyAttachmentResource,
		workflowexpiration.NewWorkflowExpirationResource,
		workflowreplication.NewWorkflowReplicationResource,
		workflowtransition.NewWorkflowTransitionResource,
	}
}

func (p *ArtescaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		accountds.NewAccountDataSource,
		accountsds.NewAccountsDataSource,
		bucketworkflowsds.NewBucketWorkflowsDataSource,
		calleridentityds.NewCallerIdentityDataSource,
		endpointsds.NewEndpointsDataSource,
		groupds.NewGroupDataSource,
		instanceds.NewInstanceDataSource,
		locationds.NewLocationDataSource,
		locationsds.NewLocationsDataSource,
		policyds.NewPolicyDataSource,
		roleds.NewRoleDataSource,
		userds.NewUserDataSource,
	}
}

func (p *ArtescaProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		assumedrolecredentials.NewAssumedRoleCredentialsEphemeralResource,
	}
}

func envOrValue(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}

func envOrDefault(val types.String, envVar, defaultVal string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultVal
}
