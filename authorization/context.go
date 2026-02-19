package authorization

import (
	"context"

	authcontext "github.com/nasermirzaei89/scribble/authentication/context"
)

// subjectFromContext returns the current subject (user ID or Anonymous) stored
// in ctx.
func subjectFromContext(ctx context.Context) string {
	return authcontext.GetSubject(ctx)
}
