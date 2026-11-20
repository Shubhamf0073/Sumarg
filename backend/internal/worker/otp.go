// Package worker runs the background jobs of `schoolfleet serve --mode=worker` (docs/02 A4) on the platform pool:
// so far the parent login-code SMS. M4.4 adds FCM pushes to the same outbox loop.
package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/shared/otp"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
)

// InPlatform runs fn in one platform-pool (BYPASSRLS) transaction; main binds it to platformdb.InPlatform.
type InPlatform func(ctx context.Context, fn func(pgx.Tx) error) error

const (
	otpBatch       = 10
	otpMaxAttempts = 3
	// otpLease is how long a claimed row is held for this worker; longer than a whole round (parallel sends, each
	// within smsTimeout), so no other worker picks it up mid-send.
	otpLease = 2 * time.Minute
	// OTPPoll is how often the outbox is checked: a parent waits at most about this long plus the gateway.
	OTPPoll = time.Second
	// otpPurgeEvery: expired challenges and settled code SMS rows are deleted after 24 h (docs/07 §3).
	otpPurgeEvery = time.Hour
)

// otpRetryDelay after the nth failed send (n = attempts so far, from 0).
func otpRetryDelay(n int16) time.Duration {
	return []time.Duration{5 * time.Second, 15 * time.Second}[min(n, 1)]
}

// OTPSender sends queued login codes (notifications event 'otp', channel 'sms').
type OTPSender struct {
	DB   InPlatform
	Keys otp.Keys
	SMS  SMSSender
	Log  *slog.Logger
	Poll time.Duration
}

// Run polls until ctx ends, and purges old rows hourly. A failed round (database down) is logged and tried again
// on the next tick.
func (w OTPSender) Run(ctx context.Context) error {
	poll := w.Poll
	if poll <= 0 {
		poll = OTPPoll
	}
	t := time.NewTicker(poll)
	defer t.Stop()
	purge := time.NewTicker(otpPurgeEvery)
	defer purge.Stop()
	w.logErr(ctx, "otp purge failed", w.Purge(ctx))
	for {
		w.logErr(ctx, "otp sms round failed", w.Drain(ctx))
		select {
		case <-ctx.Done():
			return nil
		case <-purge.C:
			w.logErr(ctx, "otp purge failed", w.Purge(ctx))
		case <-t.C:
		}
	}
}

func (w OTPSender) logErr(ctx context.Context, msg string, err error) {
	if err != nil && ctx.Err() == nil {
		w.Log.ErrorContext(ctx, msg, slog.Any("error", err))
	}
}

// Purge deletes login-code rows older than 24 h in every school.
func (w OTPSender) Purge(ctx context.Context) error {
	return w.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		c, err := q.PurgeOTPChallenges(ctx)
		if err != nil {
			return err
		}
		n, err := q.PurgeOTPNotifications(ctx)
		if err != nil {
			return err
		}
		if c+n > 0 {
			w.Log.InfoContext(ctx, "otp purge", slog.Int64("challenges", c), slog.Int64("notifications", n))
		}
		return nil
	})
}

// Drain settles every due row, a batch at a time.
func (w OTPSender) Drain(ctx context.Context) error {
	for {
		n, err := w.round(ctx)
		if err != nil || n < otpBatch {
			return err
		}
	}
}

// round claims up to otpBatch rows in a short transaction, then sends them in parallel with no transaction open and
// settles each in its own: a slow gateway holds no locks or connections, and one row's failure never rolls back
// another's settled state.
func (w OTPSender) round(ctx context.Context) (int, error) {
	var rows []platformq.ClaimOTPSMSRow
	err := w.DB(ctx, func(tx pgx.Tx) error {
		var err error
		rows, err = platformq.New(tx).ClaimOTPSMS(ctx, otpLease.Seconds(), otpBatch)
		return err
	})
	if err != nil {
		return 0, err
	}
	errs := make([]error, len(rows))
	var wg sync.WaitGroup
	for i, r := range rows {
		wg.Go(func() { errs[i] = w.settle(ctx, r) })
	}
	wg.Wait()
	return len(rows), errors.Join(errs...)
}

// settle decides one claimed row: skip, send, retry or fail. Each outcome is written in its own transaction.
func (w OTPSender) settle(ctx context.Context, r platformq.ClaimOTPSMSRow) error {
	write := func(fn func(q *platformq.Queries) (int64, error)) error {
		return w.DB(ctx, func(tx pgx.Tx) error { return oneRow(fn(platformq.New(tx))) })
	}
	done := func(status string, reason *string) error {
		return write(func(q *platformq.Queries) (int64, error) {
			return q.SettleNotification(ctx, status, reason, r.TenantID, r.ID)
		})
	}
	skip := func(reason string) error { return done("skipped", &reason) }
	switch {
	case r.Stale:
		return skip("expired before sending")
	case r.IsDemo:
		return skip("demo school: no SMS to real numbers")
	case !r.AppEnabled:
		return skip("guardian app disabled")
	}
	code, err := w.Keys.Open(r.Box, otp.BoxAAD(r.TenantID, r.DedupeKey))
	if err != nil {
		reason := "sealed code does not open"
		w.Log.ErrorContext(ctx, "otp sms: "+reason, slog.Int64("tenant_id", r.TenantID), slog.Int64("notification_id", r.ID))
		return done("failed", &reason)
	}
	sctx, cancel := context.WithTimeout(ctx, smsTimeout)
	sendErr := w.SMS.SendOTP(sctx, OTPSMS{Mobile: r.Mobile, Code: code, School: r.ShortName})
	cancel()
	if sendErr == nil {
		return done("sent", nil)
	}
	msg := sendErr.Error()
	w.Log.WarnContext(ctx, "otp sms send failed", slog.Int64("tenant_id", r.TenantID),
		slog.Int64("notification_id", r.ID), slog.String("error", msg))
	if r.Attempts+1 >= otpMaxAttempts {
		return done("failed", &msg)
	}
	return write(func(q *platformq.Queries) (int64, error) {
		return q.RetryNotification(ctx, msg, otpRetryDelay(r.Attempts).Seconds(), r.TenantID, r.ID)
	})
}

// errNotQueued: a claimed row was no longer queued (only one worker holds a row's lease, so this means a bug).
var errNotQueued = errors.New("worker: claimed notification is not queued")

func oneRow(n int64, err error) error {
	if err == nil && n != 1 {
		return errNotQueued
	}
	return err
}
