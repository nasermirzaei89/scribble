package discuss_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fileadapter "github.com/casbin/casbin/v3/persist/file-adapter"
	"github.com/google/uuid"
	authcontext "github.com/nasermirzaei89/scribble/authentication/context"
	"github.com/nasermirzaei89/scribble/authorization"
	"github.com/nasermirzaei89/scribble/authorization/casbin"
	"github.com/nasermirzaei89/scribble/discuss"
	"github.com/stretchr/testify/require"
)

type stubService struct{}

func (s *stubService) CreateComment(ctx context.Context, req discuss.CreateCommentRequest) (*discuss.Comment, error) {
	return &discuss.Comment{
		ID:       uuid.NewString(),
		PostID:   req.PostID,
		AuthorID: req.AuthorID,
		Content:  req.Content,
	}, nil
}

func (s *stubService) ListComments(ctx context.Context, postID string) ([]*discuss.Comment, error) {
	return []*discuss.Comment{}, nil
}

func (s *stubService) CountComments(ctx context.Context, postID string) (int, error) {
	return 0, nil
}

func TestAuthorizationMiddleware(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.csv")
	content := []byte(`g, system:anonymous, system:unauthenticated

p, system:authenticated, github.com/nasermirzaei89/scribble/discuss, *, createComment
p, system:authenticated, github.com/nasermirzaei89/scribble/discuss, *, listComments
p, system:unauthenticated, github.com/nasermirzaei89/scribble/discuss, *, listComments
p, system:authenticated, github.com/nasermirzaei89/scribble/discuss, *, countComments
p, system:unauthenticated, github.com/nasermirzaei89/scribble/discuss, *, countComments
`)

	err := os.WriteFile(tmpFile, content, 0o600)
	require.NoError(t, err)

	adapter := fileadapter.NewAdapter(tmpFile)

	provider, err := casbin.NewAuthorizationProvider(adapter)
	require.NoError(t, err)

	authzSvc, err := authorization.NewService(provider)
	require.NoError(t, err)

	client := authorization.NewClient(authzSvc)
	svc := discuss.NewAuthorizationMiddleware(client, &stubService{})

	userID := uuid.NewString()
	err = client.AddToGroup(ctx, userID, authcontext.Authenticated)
	require.NoError(t, err)

	authorID := uuid.NewString()
	postID := uuid.NewString()

	anonymousCtx := ctx
	authenticatedCtx := authcontext.WithSubject(ctx, userID)

	t.Run("anonymous", func(t *testing.T) {
		_, err := svc.CreateComment(anonymousCtx, discuss.CreateCommentRequest{
			PostID:   postID,
			AuthorID: authorID,
			Content:  "comment",
		})
		require.Error(t, err)

		accessDeniedErr := &authorization.AccessDeniedError{}
		require.ErrorAs(t, err, &accessDeniedErr)

		_, err = svc.ListComments(anonymousCtx, postID)
		require.NoError(t, err)

		_, err = svc.CountComments(anonymousCtx, postID)
		require.NoError(t, err)
	})

	t.Run("authenticated", func(t *testing.T) {
		_, err := svc.CreateComment(authenticatedCtx, discuss.CreateCommentRequest{
			PostID:   postID,
			AuthorID: authorID,
			Content:  "comment",
		})
		require.NoError(t, err)

		_, err = svc.ListComments(authenticatedCtx, postID)
		require.NoError(t, err)

		_, err = svc.CountComments(authenticatedCtx, postID)
		require.NoError(t, err)
	})
}
