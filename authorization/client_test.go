package authorization_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fileadapter "github.com/casbin/casbin/v3/persist/file-adapter"
	stringadapter "github.com/casbin/casbin/v3/persist/string-adapter"
	authcontext "github.com/nasermirzaei89/scribble/authentication/context"
	"github.com/nasermirzaei89/scribble/authorization"
	"github.com/nasermirzaei89/scribble/authorization/casbin"
	"github.com/stretchr/testify/require"
)

func TestClient_CheckAccess(t *testing.T) {
	ctx := context.Background()

	adapter := stringadapter.NewAdapter(`p, group1, domain1, data1, read
p, group1, domain1, data2, write
g, alice, group1
`)

	casbinProvider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(casbinProvider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)

	t.Run("allowed access", func(t *testing.T) {
		err = client.CheckAccess(authcontext.WithSubject(ctx, "alice"), "domain1", "data1", "read")
		require.NoError(t, err)
	})

	t.Run("denied access", func(t *testing.T) {
		err = client.CheckAccess(authcontext.WithSubject(ctx, "alice"), "domain1", "data2", "read")
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)
	})

	t.Run("another user", func(t *testing.T) {
		err = client.CheckAccess(authcontext.WithSubject(ctx, "bob"), "domain1", "data1", "read")
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)
	})

	t.Run("anonymous access", func(t *testing.T) {
		err = client.CheckAccess(ctx, "domain1", "data1", "read")
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)
	})
}

func TestClient_CanI(t *testing.T) {
	ctx := context.Background()

	adapter := stringadapter.NewAdapter(`p, group1, domain1, data1, read
p, group1, domain1, data2, write
g, alice, group1
`)

	casbinProvider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(casbinProvider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)

	t.Run("allowed access", func(t *testing.T) {
		allowed := client.CanI(authcontext.WithSubject(ctx, "alice"), "domain1", "data1", "read")
		require.True(t, allowed)
	})

	t.Run("denied access", func(t *testing.T) {
		allowed := client.CanI(authcontext.WithSubject(ctx, "alice"), "domain1", "data2", "read")
		require.False(t, allowed)
	})

	t.Run("another user", func(t *testing.T) {
		allowed := client.CanI(authcontext.WithSubject(ctx, "bob"), "domain1", "data1", "read")
		require.False(t, allowed)
	})

	t.Run("anonymous access", func(t *testing.T) {
		allowed := client.CanI(ctx, "domain1", "data1", "read")
		require.False(t, allowed)
	})
}

func TestClient_Can(t *testing.T) {
	ctx := context.Background()

	adapter := stringadapter.NewAdapter(`p, group1, domain1, data1, read
p, group1, domain1, data2, write
g, alice, group1
`)

	casbinProvider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(casbinProvider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)

	t.Run("allowed access", func(t *testing.T) {
		allowed := client.Can(ctx, "alice", "domain1", "data1", "read")
		require.True(t, allowed)
	})

	t.Run("denied access", func(t *testing.T) {
		allowed := client.Can(ctx, "alice", "domain1", "data2", "read")
		require.False(t, allowed)
	})

	t.Run("another user", func(t *testing.T) {
		allowed := client.Can(ctx, "bob", "domain1", "data1", "read")
		require.False(t, allowed)
	})
}

func TestClient_AddPolicyForSubject(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.csv")
	content := []byte("p, group1, domain1, data1, read")

	err := os.WriteFile(tmpFile, content, 0o600)
	require.NoError(t, err)

	// needs to use file adapter, because string adapter doesn't support
	// github.com/casbin/casbin/v3/persist.BatchAdapter
	adapter := fileadapter.NewAdapter(tmpFile)

	casbinProvider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(casbinProvider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)

	t.Run("add policy and check access", func(t *testing.T) {
		err = client.AddPolicyForSubject(ctx, "alice", "domain1", "data1", "read")
		require.NoError(t, err)

		err = client.CheckAccess(authcontext.WithSubject(ctx, "alice"), "domain1", "data1", "read")
		require.NoError(t, err)

		err = client.CheckAccess(authcontext.WithSubject(ctx, "alice"), "domain1", "data1", "write")
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)

		err = client.CheckAccess(authcontext.WithSubject(ctx, "bob"), "domain1", "data1", "read")
		require.Error(t, err)

		accessDeniedErr = &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)
	})
}

func TestClient_AddToGroup(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.csv")
	content := []byte("p, group1, domain1, data1, read")

	err := os.WriteFile(tmpFile, content, 0o600)
	require.NoError(t, err)

	adapter := fileadapter.NewAdapter(tmpFile)

	casbinProvider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(casbinProvider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)

	t.Run("add to group and check access", func(t *testing.T) {
		err = client.AddToGroup(ctx, "alice", "group1")
		err = client.AddToGroup(ctx, "bob", "group2")
		require.NoError(t, err)

		err = client.CheckAccess(authcontext.WithSubject(ctx, "alice"), "domain1", "data1", "read")
		require.NoError(t, err)

		err = client.CheckAccess(authcontext.WithSubject(ctx, "bob"), "domain1", "data1", "read")
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)
	})
}
