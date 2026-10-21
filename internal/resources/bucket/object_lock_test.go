package bucket

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func bucketValue(t *testing.T, r *BucketResource, values map[string]tftypes.Value) (tftypes.Value, tfsdk.Plan) {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	all := map[string]tftypes.Value{
		"name":                tftypes.NewValue(tftypes.String, "bkt"),
		"location_constraint": tftypes.NewValue(tftypes.String, nil),
		"versioning_enabled":  tftypes.NewValue(tftypes.Bool, nil),
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, nil),
		"account_name":        tftypes.NewValue(tftypes.String, "app"),
		"force_destroy":       tftypes.NewValue(tftypes.Bool, nil),
	}
	for k, v := range values {
		all[k] = v
	}
	v := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), all)
	return v, tfsdk.Plan{Schema: schemaResp.Schema, Raw: v}
}

func TestCreateWithObjectLock(t *testing.T) {
	var lockHeader string
	r := newTestBucketResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/bkt" && req.URL.RawQuery == "":
			lockHeader = req.Header.Get("X-Amz-Bucket-Object-Lock-Enabled")
			if !strings.Contains(req.Header.Get("Authorization"), "x-amz-bucket-object-lock-enabled") {
				t.Errorf("object lock header not signed: %s", req.Header.Get("Authorization"))
			}
		case req.Method == http.MethodPut:
			t.Errorf("unexpected %s %s: versioning is already enabled by object lock", req.Method, req.URL)
		}
	})
	_, plan := bucketValue(t, r, map[string]tftypes.Value{
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, true),
		"versioning_enabled":  tftypes.NewValue(tftypes.Bool, true),
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Raw.Type(), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if lockHeader != "true" {
		t.Errorf("x-amz-bucket-object-lock-enabled = %q, want true", lockHeader)
	}
	var got BucketResourceModel
	resp.State.Get(context.Background(), &got)
	if !got.ObjectLockEnabled.ValueBool() || !got.VersioningEnabled.ValueBool() {
		t.Errorf("state = %+v, want object lock and versioning enabled", got)
	}
}

func TestCreateWithoutObjectLockUnset(t *testing.T) {
	r := newTestBucketResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Amz-Bucket-Object-Lock-Enabled") != "" {
			t.Errorf("object lock header sent for a bucket without object lock")
		}
	})
	_, plan := bucketValue(t, r, map[string]tftypes.Value{
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"versioning_enabled":  tftypes.NewValue(tftypes.Bool, false),
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Raw.Type(), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	var got BucketResourceModel
	resp.State.Get(context.Background(), &got)
	if got.ObjectLockEnabled.IsUnknown() || got.ObjectLockEnabled.ValueBool() {
		t.Errorf("object_lock_enabled = %v, want false", got.ObjectLockEnabled)
	}
}

func TestReadObjectLockEnabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
		want bool
	}{
		{"no object lock", `<Error><Code>ObjectLockConfigurationNotFoundError</Code><Message>The object lock configuration was not found</Message></Error>`, http.StatusNotFound, false},
		{"object lock", `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`, http.StatusOK, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestBucketResource(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.URL.RawQuery {
				case "object-lock=":
					w.WriteHeader(tc.code)
					_, _ = w.Write([]byte(tc.body))
				case "location=":
					_, _ = w.Write([]byte(`<LocationConstraint></LocationConstraint>`))
				case "versioning=":
					_, _ = w.Write([]byte(`<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`))
				}
			})
			v, plan := bucketValue(t, r, map[string]tftypes.Value{
				"versioning_enabled":  tftypes.NewValue(tftypes.Bool, true),
				"object_lock_enabled": tftypes.NewValue(tftypes.Bool, nil),
			})
			state := tfsdk.State{Schema: plan.Schema, Raw: v}
			resp := &resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			var got BucketResourceModel
			resp.State.Get(context.Background(), &got)
			if got.ObjectLockEnabled.ValueBool() != tc.want {
				t.Errorf("object_lock_enabled = %v, want %v", got.ObjectLockEnabled, tc.want)
			}
		})
	}
}

func TestValidateConfigObjectLockNeedsVersioning(t *testing.T) {
	r := NewBucketResource().(*BucketResource)
	for _, tc := range []struct {
		versioning any
		wantErr    bool
	}{
		{false, true},
		{true, false},
		{nil, false},
	} {
		v, plan := bucketValue(t, r, map[string]tftypes.Value{
			"object_lock_enabled": tftypes.NewValue(tftypes.Bool, true),
			"versioning_enabled":  tftypes.NewValue(tftypes.Bool, tc.versioning),
		})
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: plan.Schema, Raw: v}}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("versioning_enabled = %v: error = %v, want %v", tc.versioning, resp.Diagnostics, tc.wantErr)
		}
	}
}

// fakeLockedBucketS3 serves a bucket with one GOVERNANCE and one COMPLIANCE
// version; only the GOVERNANCE one can be deleted, and only with bypass.
func fakeLockedBucketS3(t *testing.T, versions map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Query().Has("versions"):
			body := `<ListVersionsResult><IsTruncated>false</IsTruncated>`
			for v := range versions {
				body += `<Version><Key>obj</Key><VersionId>` + v + `</VersionId></Version>`
			}
			_, _ = w.Write([]byte(body + `</ListVersionsResult>`))
		case req.Method == http.MethodGet && req.URL.Query().Has("retention"):
			mode := versions[req.URL.Query().Get("versionId")]
			_, _ = w.Write([]byte(`<Retention><Mode>` + mode + `</Mode><RetainUntilDate>2030-01-01T00:00:00.000Z</RetainUntilDate></Retention>`))
		case req.Method == http.MethodDelete && req.URL.Path == "/bkt/obj":
			v := req.URL.Query().Get("versionId")
			bypass := req.Header.Get("X-Amz-Bypass-Governance-Retention") == "true"
			if versions[v] == "COMPLIANCE" || (versions[v] == "GOVERNANCE" && !bypass) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
				return
			}
			delete(versions, v)
			w.WriteHeader(http.StatusNoContent)
		case req.Method == http.MethodDelete && req.URL.Path == "/bkt":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", req.Method, req.URL)
		}
	}
}

func TestForceDestroyBypassesGovernance(t *testing.T) {
	versions := map[string]string{"g1": "GOVERNANCE"}
	resp := deleteBucket(t, newTestBucketResource(t, fakeLockedBucketS3(t, versions)), true, true)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if len(versions) != 0 {
		t.Errorf("versions left: %v", versions)
	}
}

func TestForceDestroyReportsComplianceRetention(t *testing.T) {
	versions := map[string]string{"c1": "COMPLIANCE"}
	resp := deleteBucket(t, newTestBucketResource(t, fakeLockedBucketS3(t, versions)), true, true)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for a version under COMPLIANCE retention")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, `"c1"`) || !strings.Contains(detail, "COMPLIANCE") || !strings.Contains(detail, "2030-01-01") {
		t.Errorf("detail = %q, want the version, mode and retain-until date", detail)
	}
}

// State from before object_lock_enabled existed has it null; an update must
// fill it from the bucket instead of saving an unknown value.
func TestUpdateFillsObjectLockFromPreAttributeState(t *testing.T) {
	for _, tc := range []struct {
		name    string
		planned any
		wantErr bool
	}{
		{"unknown in plan", tftypes.UnknownValue, false},
		{"matches bucket", true, false},
		{"differs from bucket", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestBucketResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.URL.RawQuery == "object-lock=" {
					_, _ = w.Write([]byte(`<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`))
				}
			})
			stateVal, _ := bucketValue(t, r, map[string]tftypes.Value{
				"versioning_enabled": tftypes.NewValue(tftypes.Bool, true),
			})
			_, plan := bucketValue(t, r, map[string]tftypes.Value{
				"versioning_enabled":  tftypes.NewValue(tftypes.Bool, true),
				"object_lock_enabled": tftypes.NewValue(tftypes.Bool, tc.planned),
			})
			state := tfsdk.State{Schema: plan.Schema, Raw: stateVal}
			resp := &resource.UpdateResponse{State: state}
			r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("diagnostics = %v, want error %v", resp.Diagnostics, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			var got BucketResourceModel
			resp.State.Get(context.Background(), &got)
			if got.ObjectLockEnabled.IsUnknown() || !got.ObjectLockEnabled.ValueBool() {
				t.Errorf("object_lock_enabled = %v, want true", got.ObjectLockEnabled)
			}
		})
	}
}

func TestObjectLockChangedIgnoresPreAttributeState(t *testing.T) {
	for _, tc := range []struct {
		name         string
		state, plan  types.Bool
		wantReplaced bool
	}{
		{"changed", types.BoolValue(false), types.BoolValue(true), true},
		{"state predates attribute", types.BoolNull(), types.BoolValue(true), false},
		{"plan unknown", types.BoolValue(true), types.BoolUnknown(), false},
	} {
		resp := &boolplanmodifier.RequiresReplaceIfFuncResponse{}
		objectLockChanged(context.Background(), planmodifier.BoolRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
		if resp.RequiresReplace != tc.wantReplaced {
			t.Errorf("%s: RequiresReplace = %v, want %v", tc.name, resp.RequiresReplace, tc.wantReplaced)
		}
	}
}
