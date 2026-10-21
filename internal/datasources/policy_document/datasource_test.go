package policydocument

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strs(values ...string) []types.String {
	out := make([]types.String, 0, len(values))
	for _, v := range values {
		out = append(out, types.StringValue(v))
	}
	return out
}

// read runs Read on the config built from m and returns the json attribute
// or the first error.
func read(t *testing.T, m model) (string, string) {
	t.Helper()
	ctx := context.Background()
	d := NewPolicyDocumentDataSource()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	m.JSON = types.StringNull()
	if m.Version.IsUnknown() || (m.Version == types.String{}) {
		m.Version = types.StringNull()
	}
	for i := range m.Statement {
		if (m.Statement[i].Sid == types.String{}) {
			m.Statement[i].Sid = types.StringNull()
		}
		if (m.Statement[i].Effect == types.String{}) {
			m.Statement[i].Effect = types.StringNull()
		}
	}
	built := tfsdk.State{Schema: schemaResp.Schema}
	if diags := built.Set(ctx, &m); diags.HasError() {
		t.Fatalf("building config: %v", diags)
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: built.Raw}}, resp)
	if resp.Diagnostics.HasError() {
		return "", resp.Diagnostics.Errors()[0].Detail()
	}
	var got model
	resp.State.Get(ctx, &got)
	return got.JSON.ValueString(), ""
}

func TestReadBuildsDocument(t *testing.T) {
	out, errMsg := read(t, model{Statement: []statementModel{
		{
			Sid:       types.StringValue("ReadData"),
			Actions:   strs("s3:GetObject", "s3:ListBucket"),
			Resources: strs("arn:aws:s3:::data", "arn:aws:s3:::data/*"),
			Principals: []principalModel{
				{Type: types.StringValue("AWS"), Identifiers: strs("arn:aws:iam::111111111111:root")},
				{Type: types.StringValue("AWS"), Identifiers: strs("arn:aws:iam::222222222222:root")},
			},
			Condition: []conditionModel{
				{Test: types.StringValue("IpAddress"), Variable: types.StringValue("aws:SourceIp"), Values: strs("10.0.0.0/8")},
			},
		},
		{
			Effect:    types.StringValue("Deny"),
			Actions:   strs("s3:DeleteBucket"),
			Resources: strs("arn:aws:s3:::data"),
			Principals: []principalModel{
				{Type: types.StringValue("*"), Identifiers: strs("*")},
			},
		},
	}})
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	want := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ReadData",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::111111111111:root",
          "arn:aws:iam::222222222222:root"
        ]
      },
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::data",
        "arn:aws:s3:::data/*"
      ],
      "Condition": {
        "IpAddress": {
          "aws:SourceIp": "10.0.0.0/8"
        }
      }
    },
    {
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:DeleteBucket",
      "Resource": "arn:aws:s3:::data"
    }
  ]
}`
	if out != want {
		t.Errorf("json =\n%s\nwant\n%s", out, want)
	}
}

func TestReadMergesSourceAndOverride(t *testing.T) {
	source := `{"Version":"2012-10-17","Statement":[
		{"Sid":"A","Effect":"Allow","Action":"s3:GetObject","Resource":"*"},
		{"Sid":"B","Effect":"Allow","Action":["s3:PutObject"],"Resource":"*"},
		{"Effect":"Allow","Action":"s3:ListAllMyBuckets","Resource":"*"}]}`
	override := `{"Statement":{"Sid":"B","Effect":"Deny","Action":"s3:PutObject","Resource":"*"}}`
	out, errMsg := read(t, model{
		SourcePolicyDocuments:   strs(source),
		OverridePolicyDocuments: strs(override, `{"Statement":[{"Sid":"New","Effect":"Allow","Action":"s3:GetBucketTagging","Resource":"*"}]}`),
		Statement: []statementModel{
			{Sid: types.StringValue("A"), Actions: strs("s3:GetObjectVersion"), Resources: strs("*")},
		},
	})
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	var doc struct {
		Statement []struct {
			Sid    string
			Effect string
			Action any
		}
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, s := range doc.Statement {
		got = append(got, s.Sid+":"+s.Effect+":"+s.Action.(string))
	}
	want := []string{
		"A:Allow:s3:GetObjectVersion", // configured statement replaces the source one
		"B:Deny:s3:PutObject",         // override replaces by Sid
		":Allow:s3:ListAllMyBuckets",  // source statement without Sid kept
		"New:Allow:s3:GetBucketTagging",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("statements = %v, want %v", got, want)
	}
}

func TestReadErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    model
		want string
	}{
		{"duplicate Sid across sources", model{SourcePolicyDocuments: strs(
			`{"Statement":[{"Sid":"A","Effect":"Allow","Action":"s3:*","Resource":"*"}]}`,
			`{"Statement":[{"Sid":"A","Effect":"Deny","Action":"s3:*","Resource":"*"}]}`)}, `duplicate Sid "A"`},
		{"duplicate configured Sid", model{Statement: []statementModel{
			{Sid: types.StringValue("A"), Actions: strs("s3:*")},
			{Sid: types.StringValue("A"), Actions: strs("s3:*")}}}, `duplicate statement Sid "A"`},
		{"invalid JSON", model{SourcePolicyDocuments: strs(`{"Statement":`)}, "parsing policy document"},
		{"unknown statement key", model{SourcePolicyDocuments: strs(`{"Statement":[{"Effect":"Allow","Actions":"s3:*"}]}`)}, `unknown field "Actions"`},
		{"anonymous principal mixed with others", model{Statement: []statementModel{{Actions: strs("s3:*"), Principals: []principalModel{
			{Type: types.StringValue("*"), Identifiers: strs("*")},
			{Type: types.StringValue("AWS"), Identifiers: strs("arn:aws:iam::111111111111:root")}}}}}, `type "*" must be the only one`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, errMsg := read(t, tc.m)
			if !strings.Contains(errMsg, tc.want) {
				t.Errorf("error = %q, want it to contain %q", errMsg, tc.want)
			}
		})
	}
}

func TestEmptyDocument(t *testing.T) {
	out, errMsg := read(t, model{Version: types.StringValue("2008-10-17")})
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	if out != "{\n  \"Version\": \"2008-10-17\",\n  \"Statement\": []\n}" {
		t.Errorf("json = %s", out)
	}
}

func TestValidateConfigConflicts(t *testing.T) {
	ctx := context.Background()
	d := NewPolicyDocumentDataSource().(*PolicyDocumentDataSource)
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	m := model{
		Version: types.StringNull(),
		JSON:    types.StringNull(),
		Statement: []statementModel{{
			Sid: types.StringNull(), Effect: types.StringNull(),
			Actions: strs("s3:*"), NotActions: strs("s3:Delete*"),
			Resources: strs("*"), NotResources: strs("arn:aws:s3:::x"),
			Principals:    []principalModel{{Type: types.StringValue("AWS"), Identifiers: strs("*")}},
			NotPrincipals: []principalModel{{Type: types.StringValue("AWS"), Identifiers: strs("*")}},
		}},
	}
	built := tfsdk.State{Schema: schemaResp.Schema}
	if diags := built.Set(ctx, &m); diags.HasError() {
		t.Fatal(diags)
	}
	resp := &datasource.ValidateConfigResponse{}
	d.ValidateConfig(ctx, datasource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: built.Raw}}, resp)
	if n := len(resp.Diagnostics.Errors()); n != 3 {
		t.Errorf("got %d errors, want 3: %v", n, resp.Diagnostics)
	}
}
