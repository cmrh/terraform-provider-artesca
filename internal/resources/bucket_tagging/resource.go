package buckettagging

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-artesca/internal/client"
	"github.com/scality/terraform-provider-artesca/internal/creds"
	validators "github.com/scality/terraform-provider-artesca/internal/validators"
)

var (
	_ resource.Resource                = &BucketTaggingResource{}
	_ resource.ResourceWithImportState = &BucketTaggingResource{}
)

type BucketTaggingResource struct {
	s3Client *client.S3Client
}

func NewBucketTaggingResource() resource.Resource {
	return &BucketTaggingResource{}
}

func (r *BucketTaggingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_tagging"
}

func (r *BucketTaggingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the tag set on an ARTESCA bucket. Replaces the entire tag set on each apply.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
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
			},
			"bucket_name": schema.StringAttribute{
				Description: "The name of the bucket to tag.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"tags": schema.MapAttribute{
				Description: "Map of tag key/value pairs. Replacing this map replaces the bucket's entire tag set.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *BucketTaggingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.s3Client = providerData.S3
}

func (r *BucketTaggingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketTaggingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromMap(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Putting bucket tags", map[string]any{"bucket": plan.BucketName.ValueString(), "count": len(tags)})

	if err := r.s3Client.PutBucketTagging(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.BucketName.ValueString(),
		tags,
	); err != nil {
		resp.Diagnostics.AddError("Error putting bucket tags", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketTaggingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketTaggingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := r.s3Client.GetBucketTagging(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.BucketName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket tags", err.Error())
		return
	}
	if tags == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	tagMap, diags := mapFromTags(ctx, tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Tags = tagMap
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketTaggingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketTaggingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromMap(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating bucket tags", map[string]any{"bucket": plan.BucketName.ValueString(), "count": len(tags)})

	if err := r.s3Client.PutBucketTagging(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.BucketName.ValueString(),
		tags,
	); err != nil {
		resp.Diagnostics.AddError("Error updating bucket tags", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketTaggingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketTaggingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket tags", map[string]any{"bucket": state.BucketName.ValueString()})

	if err := r.s3Client.DeleteBucketTagging(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.BucketName.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error deleting bucket tags", err.Error())
	}
}

func (r *BucketTaggingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "bucket_name", req, resp)
}

// tagsFromMap converts the Terraform map into a stable, key-sorted slice of
// BucketTag. Sorting keeps the on-wire order deterministic across applies so
// signed-request bodies remain reproducible.
func tagsFromMap(ctx context.Context, m types.Map) ([]client.BucketTag, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	raw := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &raw, false)
	if diags.HasError() {
		return nil, diags
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tags := make([]client.BucketTag, 0, len(keys))
	for _, k := range keys {
		tags = append(tags, client.BucketTag{Key: k, Value: raw[k]})
	}
	return tags, nil
}

func mapFromTags(ctx context.Context, tags []client.BucketTag) (types.Map, diag.Diagnostics) {
	elements := make(map[string]string, len(tags))
	for _, t := range tags {
		elements[t.Key] = t.Value
	}
	return types.MapValueFrom(ctx, types.StringType, elements)
}
