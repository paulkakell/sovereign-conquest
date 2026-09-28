package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Session is read from the database for every authenticated request. A JWT alone
// never grants administrator privileges or bypasses an account restriction.
type Session struct {
	IsAdmin            bool
	MustChangePassword bool
	Version            int64
	Status             string
	SuspendedUntil     *time.Time
}

func (s Session) EffectiveStatus(now time.Time) string {
	if s.Status == "suspended" && s.SuspendedUntil != nil && !s.SuspendedUntil.After(now) {
		return "active"
	}
	return s.Status
}

func LoadSession(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID, playerID string) (Session, error) {
	var session Session
	err := q.QueryRow(ctx, `
		SELECT u.is_admin, u.must_change_password, u.session_version,
		       u.account_status, u.suspended_until
		FROM users u JOIN players p ON p.user_id = u.id
		WHERE u.id = $1 AND p.id = $2
	`, userID, playerID).Scan(&session.IsAdmin, &session.MustChangePassword,
		&session.Version, &session.Status, &session.SuspendedUntil)
	return session, err
}
