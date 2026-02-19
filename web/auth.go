package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/nasermirzaei89/scribble/auth"
	authcontext "github.com/nasermirzaei89/scribble/auth/context"
)

func (h *HTTPHandler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionValueNotFoundError *SessionValueNotFoundError

		sessionId, err := h.getSessionValue(r, sessionIDKey)
		if err != nil && !errors.As(err, &sessionValueNotFoundError) {
			slog.ErrorContext(
				r.Context(),
				"error on getting session value",
				"key",
				sessionIDKey,
				"error",
				err,
			)
			http.Error(w, "error on getting session value", http.StatusInternalServerError)

			return
		}

		if sessionId != nil && sessionId.(string) != "" {
			session, err := h.authSvc.GetSession(r.Context(), sessionId.(string))
			if err != nil {
				var sessionNotFoundError *auth.SessionNotFoundError
				if errors.As(err, &sessionNotFoundError) {
					err = h.deleteSessionValue(w, r, sessionIDKey)
					if err != nil {
						slog.ErrorContext(
							r.Context(),
							"error on deleting session value",
							"key",
							sessionIDKey,
							"error",
							err,
						)
						http.Error(
							w,
							"error on deleting session value",
							http.StatusInternalServerError,
						)

						return
					}

					next.ServeHTTP(w, r)

					return
				}

				slog.ErrorContext(
					r.Context(),
					"error on getting session",
					"sessionId",
					sessionId,
					"error",
					err,
				)
				http.Error(w, "error on getting session", http.StatusInternalServerError)

				return
			}

			r = r.WithContext(authcontext.WithSessionID(r.Context(), session.ID))

			user, err := h.authSvc.GetUser(r.Context(), session.UserID)
			if err != nil {
				var userNotFoundError *auth.UserNotFoundError
				if errors.As(err, &userNotFoundError) {
					err = h.authSvc.Logout(r.Context(), session.ID)
					if err != nil {
						slog.ErrorContext(
							r.Context(),
							"error on logging out session",
							"sessionId",
							session.ID,
							"error",
							err,
						)
						http.Error(w, "error on logging out session", http.StatusInternalServerError)

						return
					}

					err = h.deleteSessionValue(w, r, sessionIDKey)
					if err != nil {
						slog.ErrorContext(
							r.Context(),
							"error on deleting session value",
							"key",
							sessionIDKey,
							"error",
							err,
						)
						http.Error(
							w,
							"error on deleting session value",
							http.StatusInternalServerError,
						)

						return
					}

					next.ServeHTTP(w, r)

					return
				}

				slog.ErrorContext(r.Context(), "error retrieving user", "error", err)
				http.Error(w, "error on retrieving user", http.StatusInternalServerError)

				return
			}

			r = r.WithContext(authcontext.WithSubject(r.Context(), user.ID))
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthenticated(r *http.Request) bool {
	return authcontext.GetSubject(r.Context()) != authcontext.Anonymous
}

func (h *HTTPHandler) AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) GuestOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)

			return
		}

		next.ServeHTTP(w, r)
	})
}
