package casbin

import (
	"context"
	"database/sql"
	"fmt"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	casbinv3 "github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
	"github.com/nasermirzaei89/scribble/authorization"
)

// rbacModel is the casbin RBAC model with resources.
const rbacModel = `
[request_definition]
r = sub, obj, res, act

[policy_definition]
p = sub, obj, res, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && (r.res == p.res || p.res == "-" || p.res == "*") && r.act == p.act
`

// AuthorizationProvider implements authorization.Provider using casbin.
type AuthorizationProvider struct {
	enforcer *casbinv3.Enforcer
}

var _ authorization.Provider = (*AuthorizationProvider)(nil)

// NewAuthorizationProvider creates a new AuthorizationProvider backed by the
// given casbin persist.Adapter.
func NewAuthorizationProvider(adapter persist.Adapter) (*AuthorizationProvider, error) {
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse casbin model: %w", err)
	}

	enforcer, err := casbinv3.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &AuthorizationProvider{enforcer: enforcer}, nil
}

// NewSQLAdapter creates a casbin sql-adapter for the given database.
func NewSQLAdapter(db *sql.DB, driverName, tableName string) (*sqladapter.Adapter, error) {
	adapter, err := sqladapter.NewAdapter(db, driverName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sql adapter: %w", err)
	}

	return adapter, nil
}

// Enforce checks whether sub may perform act on obj/res.
func (p *AuthorizationProvider) Enforce(sub, obj, res, act string) (bool, error) {
	allowed, err := p.enforcer.Enforce(sub, obj, res, act)
	if err != nil {
		return false, fmt.Errorf("failed to enforce policy: %w", err)
	}

	return allowed, nil
}

// AddGroupingPolicy adds a g(sub, group) rule to casbin.
func (p *AuthorizationProvider) AddGroupingPolicy(sub, group string) error {
	_, err := p.enforcer.AddGroupingPolicy(sub, group)
	if err != nil {
		return fmt.Errorf("failed to add grouping policy: %w", err)
	}

	return nil
}

// AddPolicyFromCSV parses casbin policy lines from a CSV string and adds them.
func (p *AuthorizationProvider) AddPolicyFromCSV(_ context.Context, content string) error {
	rules := parsePolicyCSV(content)

	for _, rule := range rules {
		if len(rule) < 2 {
			continue
		}

		switch rule[0] {
		case "p":
			_, err := p.enforcer.AddNamedPolicy("p", toInterface(rule[1:])...)
			if err != nil {
				return fmt.Errorf("failed to add policy rule %v: %w", rule, err)
			}
		case "g":
			_, err := p.enforcer.AddNamedGroupingPolicy("g", toInterface(rule[1:])...)
			if err != nil {
				return fmt.Errorf("failed to add grouping policy rule %v: %w", rule, err)
			}
		}
	}

	return nil
}

func toInterface(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}

	return out
}
