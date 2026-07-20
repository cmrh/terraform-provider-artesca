package endpoint

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &EndpointResource{}
	_ resource.ResourceWithImportState = &EndpointResource{}
)

type EndpointResource struct {
	client *client.ManagementClient
}

func NewEndpointResource() resource.Resource {
	return &EndpointResource{}
}

func (r *EndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint"
}

func (r *EndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ARTESCA data service endpoint.",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Description: "The hostname for the endpoint (e.g., 'mybucket.example.com'). Must be a valid DNS hostname or IP address.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.Hostname{},
				},
			},
			"location_name": schema.StringAttribute{
				Description: "The name of the location this endpoint points to. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"is_builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in endpoint.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = providerData.Management
}

func (r *EndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep := &client.Endpoint{
		Hostname:     plan.Hostname.ValueString(),
		LocationName: plan.LocationName.ValueString(),
	}

	tflog.Debug(ctx, "Creating endpoint", map[string]any{"hostname": ep.Hostname})

	created, err := r.client.CreateEndpoint(ctx, ep)
	if err != nil {
		resp.Diagnostics.AddError("Error creating endpoint", err.Error())
		return
	}

	plan.Hostname = types.StringValue(created.Hostname)
	plan.LocationName = types.StringValue(created.LocationName)
	plan.IsBuiltin = types.BoolValue(created.IsBuiltin)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep, err := r.client.GetEndpoint(ctx, state.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading endpoint", err.Error())
		return
	}
	if ep == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Hostname = types.StringValue(ep.Hostname)
	state.LocationName = types.StringValue(ep.LocationName)
	state.IsBuiltin = types.BoolValue(ep.IsBuiltin)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EndpointResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Endpoints are immutable (both fields are ForceNew), so Update should never be called.
	resp.Diagnostics.AddError("Update not supported", "Endpoints cannot be updated in place. Both hostname and location_name require replacement.")
}

func (r *EndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting endpoint", map[string]any{"hostname": state.Hostname.ValueString()})

	err := r.client.DeleteEndpoint(ctx, state.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting endpoint", err.Error())
		return
	}
}

func (r *EndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("hostname"), req, resp)
}
