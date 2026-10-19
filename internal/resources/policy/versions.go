package policy

import (
	"context"
	"sort"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/policydoc"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// maxPolicyVersions is how many versions ARTESCA keeps per managed policy;
// creating one more fails with LimitExceeded.
const maxPolicyVersions = 5

// oldestNonDefault returns the oldest version that isn't the default, or ""
// when there is none.
func oldestNonDefault(versions []client.PolicyVersion) string {
	candidates := make([]client.PolicyVersion, 0, len(versions))
	for _, v := range versions {
		if !v.IsDefault {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].CreateDate != candidates[j].CreateDate {
			return candidates[i].CreateDate < candidates[j].CreateDate
		}
		return candidates[i].VersionID < candidates[j].VersionID
	})
	return candidates[0].VersionID
}

// makeRoomForVersion deletes the oldest non-default version when the policy
// is at the version limit.
func (r *PolicyResource) makeRoomForVersion(ctx context.Context, creds client.Credentials, arn string) error {
	versions, err := r.iamClient.ListPolicyVersions(ctx, creds, arn)
	if err != nil {
		return err
	}
	if len(versions) < maxPolicyVersions {
		return nil
	}
	if oldest := oldestNonDefault(versions); oldest != "" {
		return r.iamClient.DeletePolicyVersion(ctx, creds, arn, oldest)
	}
	return nil
}

// deleteNonDefaultVersions removes the versions a policy must not have when
// it is deleted.
func (r *PolicyResource) deleteNonDefaultVersions(ctx context.Context, creds client.Credentials, arn string) error {
	versions, err := r.iamClient.ListPolicyVersions(ctx, creds, arn)
	if err != nil {
		return err
	}
	for _, v := range versions {
		if !v.IsDefault {
			if err := r.iamClient.DeletePolicyVersion(ctx, creds, arn, v.VersionID); err != nil {
				return err
			}
		}
	}
	return nil
}

// defaultVersionModifier keeps default_version_id unless the policy document
// changes, which creates a new default version.
type defaultVersionModifier struct{}

func (m defaultVersionModifier) Description(_ context.Context) string {
	return "Unknown when policy_document changes; otherwise the current version."
}

func (m defaultVersionModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m defaultVersionModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planned, current types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("policy_document"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("policy_document"), &current)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planned.IsUnknown() || !policydoc.Equivalent(planned.ValueString(), current.ValueString()) {
		resp.PlanValue = types.StringUnknown()
		return
	}
	resp.PlanValue = req.StateValue
}
