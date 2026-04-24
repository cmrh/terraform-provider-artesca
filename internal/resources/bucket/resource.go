package bucket

import (
	"context"
	"fmt"

<<<<<<< Updated upstream
=======
	"github.com/hashicorp/terraform-plugin-framework/path"
>>>>>>> Stashed changes
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
<<<<<<< Updated upstream
	"github.com/hashicorp/terraform-plugin-framework/types"
=======
>>>>>>> Stashed changes
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
	validators "github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

<<<<<<< Updated upstream
var _ resource.Resource = &BucketResource{}
=======
var (
	_ resource.Resource                = &BucketResource{}
	_ resource.ResourceWithImportState = &BucketResource{}
)
>>>>>>> Stashed changes

type BucketResource struct {
	s3Client *client.S3Client
}

<<<<<<< Updated upstream
type BucketResourceModel struct {
	Name               types.String `tfsdk:"name"`
	LocationConstraint types.String `tfsdk:"location_constraint"`
	AccountAccessKey   types.String `tfsdk:"account_access_key"`
	AccountSecretKey   types.String `tfsdk:"account_secret_key"`
}

=======
>>>>>>> Stashed changes
func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

func (r *BucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
<<<<<<< Updated upstream
		Description: "Manages an S3 bucket on the ARTESCA S3 endpoint.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the S3 bucket.",
=======
		Description: "Manages an S3 bucket in ARTESCA.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the bucket. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
>>>>>>> Stashed changes
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"location_constraint": schema.StringAttribute{
<<<<<<< Updated upstream
				Description: "The ARTESCA location name to use as the storage backend for this bucket.",
				Optional:    true,
=======
				Description: "The location constraint (storage location name) for the bucket.",
				Required:    true,
>>>>>>> Stashed changes
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_access_key": schema.StringAttribute{
<<<<<<< Updated upstream
				Description: "The access key of the account that owns this bucket.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account that owns this bucket.",
				Required:    true,
				Sensitive:   true,
=======
				Description: "The access key of the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
>>>>>>> Stashed changes
			},
		},
	}
}

func (r *BucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
<<<<<<< Updated upstream
	if providerData.S3 == nil {
		resp.Diagnostics.AddError(
			"S3 Client Not Configured",
			"The s3_endpoint must be set in the provider configuration or via the ARTESCA_S3_ENDPOINT environment variable to use bucket resources.",
		)
		return
	}
=======
>>>>>>> Stashed changes
	r.s3Client = providerData.S3
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.Name.ValueString()
	accessKey := plan.AccountAccessKey.ValueString()
	secretKey := plan.AccountSecretKey.ValueString()
	locationConstraint := plan.LocationConstraint.ValueString()

<<<<<<< Updated upstream
	tflog.Debug(ctx, "Creating bucket", map[string]any{
		"bucket":   bucketName,
		"location": locationConstraint,
	})
=======
	tflog.Debug(ctx, "Creating S3 bucket", map[string]any{"bucket": bucketName, "location": locationConstraint})
>>>>>>> Stashed changes

	err := r.s3Client.CreateBucket(ctx, accessKey, secretKey, bucketName, locationConstraint)
	if err != nil {
		resp.Diagnostics.AddError("Error creating bucket", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

<<<<<<< Updated upstream
	bucketName := state.Name.ValueString()
	accessKey := state.AccountAccessKey.ValueString()
	secretKey := state.AccountSecretKey.ValueString()
=======
	accessKey := state.AccountAccessKey.ValueString()
	secretKey := state.AccountSecretKey.ValueString()
	bucketName := state.Name.ValueString()
>>>>>>> Stashed changes

	exists, err := r.s3Client.HeadBucket(ctx, accessKey, secretKey, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

<<<<<<< Updated upstream
	if !state.LocationConstraint.IsNull() {
		loc, err := r.s3Client.GetBucketLocation(ctx, accessKey, secretKey, bucketName)
		if err != nil {
			resp.Diagnostics.AddError("Error reading bucket location", err.Error())
			return
		}
		if loc != "" {
			state.LocationConstraint = types.StringValue(loc)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
=======
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Buckets cannot be updated in place.")
>>>>>>> Stashed changes
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()
	accessKey := state.AccountAccessKey.ValueString()
	secretKey := state.AccountSecretKey.ValueString()

<<<<<<< Updated upstream
	tflog.Debug(ctx, "Deleting bucket", map[string]any{"bucket": bucketName})
=======
	tflog.Debug(ctx, "Deleting S3 bucket", map[string]any{"bucket": bucketName})
>>>>>>> Stashed changes

	err := r.s3Client.DeleteBucket(ctx, accessKey, secretKey, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting bucket", err.Error())
		return
	}
}
<<<<<<< Updated upstream
=======

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
>>>>>>> Stashed changes
