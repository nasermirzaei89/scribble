package casbin

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
	"github.com/nasermirzaei89/scribble/authorization"
)

const ObjectNone = "-"

//go:embed model.conf
var casbinModelContent string

type AuthorizationProvider struct {
	enforcer *casbin.Enforcer
}

func NewAuthorizationProvider(persistAdapter persist.Adapter) (*AuthorizationProvider, error) {
	// TODO: validate arguments
	casbinModel, err := model.NewModelFromString(casbinModelContent)
	if err != nil {
		return nil, fmt.Errorf("failed to load casbin model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(casbinModel, persistAdapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	enforcer.EnableAutoSave(true)
	enforcer.EnableAutoBuildRoleLinks(true)

	err = enforcer.LoadPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to load db policy: %w", err)
	}

	return &AuthorizationProvider{
		enforcer: enforcer,
	}, nil
}

func (ap *AuthorizationProvider) CheckAccess(
	ctx context.Context,
	req authorization.CheckAccessRequest,
) (*authorization.CheckAccessResponse, error) {
	if req.Object == "" {
		req.Object = ObjectNone
	}

	allowed, err := ap.enforcer.Enforce(req.Subject, req.Domain, req.Object, req.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	return &authorization.CheckAccessResponse{
		Allowed: allowed,
		Denied:  false,
		Reason:  "",
	}, nil
}

func (ap *AuthorizationProvider) AddPolicy(ctx context.Context, reqs ...authorization.AddPolicyRequest) error {
	rules := make([][]string, 0, len(reqs))

	for _, req := range reqs {
		if req.Object == "" {
			req.Object = ObjectNone
		}

		rules = append(rules, []string{req.Subject, req.Domain, req.Object, req.Action})
	}

	_, err := ap.enforcer.AddPolicies(rules)
	if err != nil {
		return fmt.Errorf("failed to add policies: %w", err)
	}

	return nil
}

func (ap *AuthorizationProvider) AddToGroup(ctx context.Context, sub string, groups ...string) error {
	rules := make([][]string, 0, len(groups))

	for _, group := range groups {
		rules = append(rules, []string{sub, group})
	}

	_, err := ap.enforcer.AddGroupingPolicies(rules)
	if err != nil {
		return fmt.Errorf("failed to add grouping policies: %w", err)
	}

	return nil
}

func (ap *AuthorizationProvider) RemovePolicy(ctx context.Context, reqs ...authorization.RemovePolicyRequest) error {
	rules := make([][]string, 0, len(reqs))

	for _, req := range reqs {
		if req.Object == "" {
			req.Object = ObjectNone
		}

		rules = append(rules, []string{req.Subject, req.Domain, req.Object, req.Action})
	}

	_, err := ap.enforcer.RemovePolicies(rules)
	if err != nil {
		return fmt.Errorf("failed to remove policies: %w", err)
	}

	return nil
}

func (ap *AuthorizationProvider) RemoveFromGroup(ctx context.Context, sub string, groups ...string) error {
	rules := make([][]string, 0, len(groups))

	for _, group := range groups {
		rules = append(rules, []string{sub, group})
	}

	_, err := ap.enforcer.RemoveGroupingPolicies(rules)
	if err != nil {
		return fmt.Errorf("failed to remove grouping policies: %w", err)
	}

	return nil
}

func (ap *AuthorizationProvider) AddPolicyFromCSV(ctx context.Context, casbinPolicyContent string) error {
	err := addPolicyFromString(ap.enforcer, casbinPolicyContent)
	if err != nil {
		return fmt.Errorf("failed to load csv policy: %w", err)
	}

	return nil
}

func addPolicyFromString(enforcer *casbin.Enforcer, policyFileContent string) error {
	reader := csv.NewReader(strings.NewReader(policyFileContent))

	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read policy content: %w", err)
	}

	for _, record := range records {
		record = normalizePolicyRecord(record)
		if len(record) == 0 || record[0] == "" {
			continue
		}

		err = addPolicyFromRecord(enforcer, record)
		if err != nil {
			return fmt.Errorf("failed to add policy from record: %w", err)
		}
	}

	return nil
}

func normalizePolicyRecord(record []string) []string {
	normalized := make([]string, len(record))
	for i := range record {
		normalized[i] = strings.TrimSpace(record[i])
	}

	return normalized
}

func addPolicyFromRecord(enforcer *casbin.Enforcer, record []string) error {
	switch record[0] {
	case "p":
		err := addPolicyIfNotExists(enforcer, record[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add policy if not exists: %w", err)
		}

	case "g":
		err := addGroupingPolicyIfNotExists(enforcer, record[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add grouping policy if not exists: %w", err)
		}
	default:
		return UnknownPolicyTypeError{PolicyType: record[0]}
	}

	return nil
}

func addPolicyIfNotExists(enforcer *casbin.Enforcer, params ...string) error {
	args := make([]any, len(params))
	for i := range args {
		args[i] = params[i]
	}

	exists, err := enforcer.HasPolicy(args...)
	if err != nil {
		return fmt.Errorf("failed to check policy: %w", err)
	}

	if exists {
		return nil
	}

	_, err = enforcer.AddPolicy(args...)
	if err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}

	return nil
}

func addGroupingPolicyIfNotExists(enforcer *casbin.Enforcer, params ...string) error {
	args := make([]any, len(params))
	for i := range args {
		args[i] = params[i]
	}

	exists, err := enforcer.HasGroupingPolicy(args...)
	if err != nil {
		return fmt.Errorf("failed to check policy: %w", err)
	}

	if exists {
		return nil
	}

	_, err = enforcer.AddGroupingPolicy(args...)
	if err != nil {
		return fmt.Errorf("failed to add grouping policy: %w", err)
	}

	return nil
}
