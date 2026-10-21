// Package policydocument implements artesca_policy_document, which builds an
// IAM policy document locally.
package policydocument

import (
	"context"
	"fmt"

	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                   = &PolicyDocumentDataSource{}
	_ datasource.DataSourceWithValidateConfig = &PolicyDocumentDataSource{}
)

const defaultVersion = "2012-10-17"

type PolicyDocumentDataSource struct{}

func NewPolicyDocumentDataSource() datasource.DataSource {
	return &PolicyDocumentDataSource{}
}

type model struct {
	Version                 types.String     `tfsdk:"version"`
	SourcePolicyDocuments   []types.String   `tfsdk:"source_policy_documents"`
	OverridePolicyDocuments []types.String   `tfsdk:"override_policy_documents"`
	Statement               []statementModel `tfsdk:"statement"`
	JSON                    types.String     `tfsdk:"json"`
}

type statementModel struct {
	Sid           types.String     `tfsdk:"sid"`
	Effect        types.String     `tfsdk:"effect"`
	Actions       []types.String   `tfsdk:"actions"`
	NotActions    []types.String   `tfsdk:"not_actions"`
	Resources     []types.String   `tfsdk:"resources"`
	NotResources  []types.String   `tfsdk:"not_resources"`
	Principals    []principalModel `tfsdk:"principals"`
	NotPrincipals []principalModel `tfsdk:"not_principals"`
	Condition     []conditionModel `tfsdk:"condition"`
}

type principalModel struct {
	Type        types.String   `tfsdk:"type"`
	Identifiers []types.String `tfsdk:"identifiers"`
}

type conditionModel struct {
	Test     types.String   `tfsdk:"test"`
	Variable types.String   `tfsdk:"variable"`
	Values   []types.String `tfsdk:"values"`
}

func (d *PolicyDocumentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_document"
}

func principalBlock(description string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Description: "Principal type, such as `AWS` or `Service`. `*` with identifiers `[\"*\"]` is everyone.",
					Required:    true,
				},
				"identifiers": schema.SetAttribute{
					Description: "Principal identifiers, such as account or role ARNs.",
					Required:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}

func (d *PolicyDocumentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	stringSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Description: description, Optional: true, ElementType: types.StringType}
	}
	resp.Schema = schema.Schema{
		Description: "Builds an IAM policy document as JSON, for artesca_policy, artesca_user_policy, artesca_group_policy, artesca_role and artesca_bucket_policy. Computed locally; no API calls.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				Description: "Policy language version: `2012-10-17` (default) or `2008-10-17`.",
				Optional:    true,
				Validators: []validator.String{
					validators.OneOf{Values: []string{"2012-10-17", "2008-10-17"}},
				},
			},
			"source_policy_documents": schema.ListAttribute{
				Description: "Policy documents (JSON) whose statements are included first. A Sid may appear in only one of them; a `statement` block with the same Sid replaces it.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"override_policy_documents": schema.ListAttribute{
				Description: "Policy documents (JSON) applied last: a statement replaces the statement with its Sid, and one without a Sid or with a new Sid is appended.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"json": schema.StringAttribute{
				Description: "The policy document as JSON.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"statement": schema.ListNestedBlock{
				Description: "A policy statement.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"sid": schema.StringAttribute{
							Description: "Statement ID.",
							Optional:    true,
						},
						"effect": schema.StringAttribute{
							Description: "`Allow` (default) or `Deny`.",
							Optional:    true,
							Validators: []validator.String{
								validators.OneOf{Values: []string{"Allow", "Deny"}},
							},
						},
						"actions":       stringSet("Actions the statement applies to, such as `s3:GetObject`. Conflicts with `not_actions`."),
						"not_actions":   stringSet("Actions the statement doesn't apply to. Conflicts with `actions`."),
						"resources":     stringSet("Resource ARNs the statement applies to. Conflicts with `not_resources`."),
						"not_resources": stringSet("Resource ARNs the statement doesn't apply to. Conflicts with `resources`."),
					},
					Blocks: map[string]schema.Block{
						"principals":     principalBlock("Principals the statement applies to (bucket policies and role trust policies). Conflicts with `not_principals`."),
						"not_principals": principalBlock("Principals the statement doesn't apply to. Conflicts with `principals`."),
						"condition": schema.ListNestedBlock{
							Description: "A condition on the statement.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"test": schema.StringAttribute{
										Description: "Condition operator, such as `StringEquals` or `IpAddress`.",
										Required:    true,
									},
									"variable": schema.StringAttribute{
										Description: "Condition key, such as `aws:SourceIp`.",
										Required:    true,
									},
									"values": schema.ListAttribute{
										Description: "Values to compare the key with; any one may match.",
										Required:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// ValidateConfig rejects statements that set both an element and its Not
// form, which the policy grammar doesn't allow.
func (d *PolicyDocumentDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config model
	// Values unknown until apply (a dynamic block, a computed list) can't be
	// read into the model; Read checks the document once they are known.
	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		return
	}
	for i, s := range config.Statement {
		p := path.Root("statement").AtListIndex(i)
		if s.Actions != nil && s.NotActions != nil {
			resp.Diagnostics.AddAttributeError(p.AtName("not_actions"), "Conflicting statement elements", "Set actions or not_actions, not both.")
		}
		if s.Resources != nil && s.NotResources != nil {
			resp.Diagnostics.AddAttributeError(p.AtName("not_resources"), "Conflicting statement elements", "Set resources or not_resources, not both.")
		}
		if len(s.Principals) > 0 && len(s.NotPrincipals) > 0 {
			resp.Diagnostics.AddAttributeError(p.AtName("not_principals"), "Conflicting statement elements", "Set principals or not_principals, not both.")
		}
	}
}

func (d *PolicyDocumentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parse := func(attr string, raws []types.String) []document {
		docs := make([]document, 0, len(raws))
		for i, raw := range raws {
			doc, err := parseDocument(raw.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root(attr).AtListIndex(i), "Invalid policy document", err.Error())
				continue
			}
			docs = append(docs, doc)
		}
		return docs
	}
	sources := parse("source_policy_documents", data.SourcePolicyDocuments)
	overrides := parse("override_policy_documents", data.OverridePolicyDocuments)
	if resp.Diagnostics.HasError() {
		return
	}

	statements := make([]statement, 0, len(data.Statement))
	for i, s := range data.Statement {
		st, err := toStatement(s)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("statement").AtListIndex(i), "Invalid statement", err.Error())
			return
		}
		statements = append(statements, st)
	}

	version := data.Version.ValueString()
	if version == "" {
		version = defaultVersion
	}
	doc, err := merge(version, sources, statements, overrides)
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy document", err.Error())
		return
	}
	out, err := encode(doc)
	if err != nil {
		resp.Diagnostics.AddError("Error encoding policy document", err.Error())
		return
	}

	data.JSON = types.StringValue(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringValues(values []types.String) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.ValueString())
	}
	return out
}

func toPrincipals(blocks []principalModel) []principal {
	out := make([]principal, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, principal{Type: b.Type.ValueString(), Identifiers: stringValues(b.Identifiers)})
	}
	return out
}

func toStatement(s statementModel) (statement, error) {
	effect := s.Effect.ValueString()
	if effect == "" {
		effect = "Allow"
	}
	st := statement{
		Sid:         s.Sid.ValueString(),
		Effect:      effect,
		Action:      stringOrList(stringValues(s.Actions)),
		NotAction:   stringOrList(stringValues(s.NotActions)),
		Resource:    stringOrList(stringValues(s.Resources)),
		NotResource: stringOrList(stringValues(s.NotResources)),
	}
	var err error
	if st.Principal, err = principalValue(toPrincipals(s.Principals)); err != nil {
		return statement{}, fmt.Errorf("principals: %w", err)
	}
	if st.NotPrincipal, err = principalValue(toPrincipals(s.NotPrincipals)); err != nil {
		return statement{}, fmt.Errorf("not_principals: %w", err)
	}
	conditions := make([]condition, 0, len(s.Condition))
	for _, c := range s.Condition {
		conditions = append(conditions, condition{Test: c.Test.ValueString(), Variable: c.Variable.ValueString(), Values: stringValues(c.Values)})
	}
	st.Condition = conditionValue(conditions)
	return st, nil
}
