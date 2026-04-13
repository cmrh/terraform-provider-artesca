package location

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LocationResourceModel struct {
	Name              types.String          `tfsdk:"name"`
	LocationType      types.String          `tfsdk:"location_type"`
	IsTransient       types.Bool            `tfsdk:"is_transient"`
	LegacyAwsBehavior types.Bool            `tfsdk:"legacy_aws_behavior"`
	SizeLimitGB       types.Int64           `tfsdk:"size_limit_gb"`
	ObjectID          types.String          `tfsdk:"object_id"`
	IsBuiltin         types.Bool            `tfsdk:"is_builtin"`
	Details           *LocationDetailsModel `tfsdk:"details"`
}

type LocationDetailsModel struct {
	AccessKey            types.String `tfsdk:"access_key"`
	SecretKey            types.String `tfsdk:"secret_key"`
	BucketName           types.String `tfsdk:"bucket_name"`
	BucketMatch          types.Bool   `tfsdk:"bucket_match"`
	Endpoint             types.String `tfsdk:"endpoint"`
	Region               types.String `tfsdk:"region"`
	ServerSideEncryption types.Bool   `tfsdk:"server_side_encryption"`
	StorageClass         types.String `tfsdk:"storage_class"`
	MpuBucketName        types.String `tfsdk:"mpu_bucket_name"`
	Username             types.String `tfsdk:"username"`
	Password             types.String `tfsdk:"password"`
	TenantName           types.String `tfsdk:"tenant_name"`
	SubscriptionID       types.String `tfsdk:"subscription_id"`
	ResourceGroup        types.String `tfsdk:"resource_group"`
	StorageAccountName   types.String `tfsdk:"storage_account_name"`
	StorageContainerName types.String `tfsdk:"storage_container_name"`
	NsID                 types.String `tfsdk:"ns_id"`
	RepoID               types.List   `tfsdk:"repo_id"`
	ProxyPath            types.String `tfsdk:"proxy_path"`
	BootstrapList        types.List   `tfsdk:"bootstrap_list"`
	ChordCos             types.Int64  `tfsdk:"chord_cos"`
	CodingParts          types.Int64  `tfsdk:"coding_parts"`
	DataParts            types.Int64  `tfsdk:"data_parts"`
	GcpEndpoint          types.String `tfsdk:"gcp_endpoint"`
	BucketPrefix         types.String `tfsdk:"bucket_prefix"`
}
