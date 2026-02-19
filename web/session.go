package web

import (
	"fmt"
	"net/http"
)

const sessionIDKey = "sessionId"

type SessionValueNotFoundError struct {
	Key string
}

func (err SessionValueNotFoundError) Error() string {
	return fmt.Sprintf("session value for key '%s' not found", err.Key)
}

func (h *HTTPHandler) getSessionValue(r *http.Request, key string) (any, error) {
	session, err := h.cookieStore.Get(r, h.sessionName)
	if err != nil {
		return nil, fmt.Errorf("error getting session: %w", err)
	}

	value, ok := session.Values[key]
	if !ok {
		return nil, &SessionValueNotFoundError{Key: key}
	}

	return value, nil
}

func (h *HTTPHandler) setSessionValue(
	w http.ResponseWriter,
	r *http.Request,
	key string,
	value any,
) error {
	session, err := h.cookieStore.Get(r, h.sessionName)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	session.Values[key] = value

	err = session.Save(r, w)
	if err != nil {
		return fmt.Errorf("error saving session: %w", err)
	}

	return nil
}

func (h *HTTPHandler) deleteSessionValue(w http.ResponseWriter, r *http.Request, key string) error {
	session, err := h.cookieStore.Get(r, h.sessionName)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	delete(session.Values, key)

	err = session.Save(r, w)
	if err != nil {
		return fmt.Errorf("error saving session: %w", err)
	}

	return nil
}
