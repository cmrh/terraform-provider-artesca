package workflowreplication

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
)

// locationTypeCRR is the location type whose destination needs a role on
// the remote site.
const locationTypeCRR = "location-scality-crr-v1"

// splitRole returns the two parts of a bucket replication Role: the source
// role and, when the configuration targets a CRR location, the destination
// role after the comma.
func splitRole(role string) (source, destination string) {
	source, destination, _ = strings.Cut(role, ",")
	return source, destination
}

// buildRole returns the Role for a configuration with rules. ARTESCA wants
// exactly "<source>,<destination>" while any rule targets a CRR location,
// and a single ARN otherwise. destination is the role CRR rules use; the
// source role is kept from current, or DefaultReplicationRole when there is
// none.
func buildRole(current string, rules []client.ReplicationRule, isCRR func(location string) (bool, error), destination string) (string, error) {
	source, _ := splitRole(current)
	if source == "" {
		source = client.DefaultReplicationRole
	}
	for _, rule := range rules {
		crr, err := isCRR(rule.StorageClass)
		if err != nil {
			return "", err
		}
		if crr {
			if destination == "" {
				return "", fmt.Errorf("rule %q targets the CRR location %q, but no destination role is set", rule.ID, rule.StorageClass)
			}
			return source + "," + destination, nil
		}
	}
	return source, nil
}

// crrChecker reports whether a location is a CRR location, looking each
// name up once.
func (r *WorkflowReplicationResource) crrChecker(ctx context.Context) func(string) (bool, error) {
	seen := map[string]bool{}
	return func(location string) (bool, error) {
		if location == "" {
			return false, nil
		}
		if crr, ok := seen[location]; ok {
			return crr, nil
		}
		loc, err := r.mgmt.GetLocation(ctx, location)
		if err != nil {
			return false, fmt.Errorf("reading location %q: %w", location, err)
		}
		crr := loc != nil && loc.LocationType == locationTypeCRR
		seen[location] = crr
		return crr, nil
	}
}

// otherCRRDestination returns the destination role that rules other than
// ruleID rely on: the current one, when any other rule targets a CRR
// location.
func otherCRRDestination(current string, rules []client.ReplicationRule, ruleID string, isCRR func(string) (bool, error)) (string, error) {
	_, destination := splitRole(current)
	for _, rule := range rules {
		if rule.ID == ruleID {
			continue
		}
		crr, err := isCRR(rule.StorageClass)
		if err != nil {
			return "", err
		}
		if crr {
			return destination, nil
		}
	}
	return "", nil
}
