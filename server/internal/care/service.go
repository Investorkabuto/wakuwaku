// Package care は服薬・タスク管理を扱う。医療的判断（用量計算・飲み合わせ等）は行わない。
package care

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	wakuwakuv1 "github.com/investorkabuto/wakuwaku/server/internal/gen/wakuwaku/v1"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	dateLayout       = "2006-01-02"
	maxNameLen       = 100
	maxDoseTextLen   = 100
	maxDoseLogRange  = 366 * 24 * time.Hour
	maxClockSkewLead = 10 * time.Minute
)

type Service struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

func (s *Service) ListMedications(ctx context.Context, _ *connect.Request[wakuwakuv1.ListMedicationsRequest]) (*connect.Response[wakuwakuv1.ListMedicationsResponse], error) {
	userID, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, dose_text, timings
		FROM medications
		WHERE user_id = $1 AND active
		ORDER BY created_at`, userID)
	if err != nil {
		return nil, internal(err)
	}
	meds, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*wakuwakuv1.Medication, error) {
		var m wakuwakuv1.Medication
		var timings []int16
		if err := row.Scan(&m.Id, &m.Name, &m.DoseText, &timings); err != nil {
			return nil, err
		}
		m.Timings = toTimings(timings)
		return &m, nil
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&wakuwakuv1.ListMedicationsResponse{Medications: meds}), nil
}

func (s *Service) CreateMedication(ctx context.Context, req *connect.Request[wakuwakuv1.CreateMedicationRequest]) (*connect.Response[wakuwakuv1.CreateMedicationResponse], error) {
	userID, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	doseText := strings.TrimSpace(req.Msg.GetDoseText())
	switch {
	case name == "":
		return nil, invalid("name is required")
	case utf8.RuneCountInString(name) > maxNameLen:
		return nil, invalid("name is too long")
	case utf8.RuneCountInString(doseText) > maxDoseTextLen:
		return nil, invalid("dose_text is too long")
	}
	timings, err := normalizeTimings(req.Msg.GetTimings())
	if err != nil {
		return nil, err
	}

	var id string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO medications (user_id, name, dose_text, timings)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text`, userID, name, doseText, timings).Scan(&id)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&wakuwakuv1.CreateMedicationResponse{Medication: &wakuwakuv1.Medication{
		Id:       id,
		Name:     name,
		DoseText: doseText,
		Timings:  toTimings(timings),
	}}), nil
}

func (s *Service) RecordDose(ctx context.Context, req *connect.Request[wakuwakuv1.RecordDoseRequest]) (*connect.Response[wakuwakuv1.RecordDoseResponse], error) {
	userID, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	logID, err := uuid.Parse(msg.GetDoseLogId())
	if err != nil {
		return nil, invalid("dose_log_id must be a UUID")
	}
	medID, err := uuid.Parse(msg.GetMedicationId())
	if err != nil {
		return nil, invalid("medication_id must be a UUID")
	}
	date, err := time.Parse(dateLayout, msg.GetScheduledDate())
	if err != nil {
		return nil, invalid("scheduled_date must be YYYY-MM-DD")
	}
	if !msg.GetTakenAt().IsValid() {
		return nil, invalid("taken_at is required")
	}
	takenAt := msg.GetTakenAt().AsTime()
	if takenAt.After(s.now().Add(maxClockSkewLead)) {
		return nil, invalid("taken_at is in the future")
	}

	var resp *wakuwakuv1.RecordDoseResponse
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// 再送（同じID）は記録済みの内容をそのまま返す。
		if existing, err := findDoseLog(ctx, tx, `id = $1 AND user_id = $2`, logID, userID); err == nil {
			resp = &wakuwakuv1.RecordDoseResponse{DoseLog: existing}
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		// 薬の所有確認と行ロック（同じ回の並行記録を直列化する）。
		var timings []int16
		err := tx.QueryRow(ctx, `
			SELECT timings FROM medications
			WHERE id = $1 AND user_id = $2 AND active
			FOR UPDATE`, medID, userID).Scan(&timings)
		if errors.Is(err, pgx.ErrNoRows) {
			return connect.NewError(connect.CodeNotFound, errors.New("medication not found"))
		} else if err != nil {
			return err
		}
		if !slices.Contains(timings, int16(msg.GetTiming())) {
			return invalid("timing is not scheduled for this medication")
		}

		// MED-05: 同じ回が記録済みなら、確認なしでは記録しない。
		if !msg.GetConfirmDuplicate() {
			existing, err := findDoseLog(ctx, tx,
				`medication_id = $1 AND scheduled_date = $2 AND timing = $3 ORDER BY taken_at LIMIT 1`,
				medID, date, int16(msg.GetTiming()))
			if err == nil {
				resp = &wakuwakuv1.RecordDoseResponse{AlreadyRecorded: true, DoseLog: existing}
				return nil
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO dose_logs (id, user_id, medication_id, timing, scheduled_date, taken_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			logID, userID, medID, int16(msg.GetTiming()), date, takenAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			// 他ユーザーが同じIDを使っている。
			return connect.NewError(connect.CodeAlreadyExists, errors.New("dose_log_id already used"))
		}
		resp = &wakuwakuv1.RecordDoseResponse{DoseLog: &wakuwakuv1.DoseLog{
			Id:            logID.String(),
			MedicationId:  medID.String(),
			Timing:        msg.GetTiming(),
			ScheduledDate: date.Format(dateLayout),
			TakenAt:       timestamppb.New(takenAt),
		}}
		return nil
	})
	if err != nil {
		return nil, asConnectError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) ListDoseLogs(ctx context.Context, req *connect.Request[wakuwakuv1.ListDoseLogsRequest]) (*connect.Response[wakuwakuv1.ListDoseLogsResponse], error) {
	userID, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	from, err1 := time.Parse(dateLayout, req.Msg.GetFromDate())
	to, err2 := time.Parse(dateLayout, req.Msg.GetToDate())
	switch {
	case err1 != nil || err2 != nil:
		return nil, invalid("from_date and to_date must be YYYY-MM-DD")
	case to.Before(from):
		return nil, invalid("to_date must not be before from_date")
	case to.Sub(from) > maxDoseLogRange:
		return nil, invalid("date range is too long")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+doseLogColumns+`
		FROM dose_logs
		WHERE user_id = $1 AND scheduled_date BETWEEN $2 AND $3
		ORDER BY scheduled_date, timing, taken_at`, userID, from, to)
	if err != nil {
		return nil, internal(err)
	}
	logs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*wakuwakuv1.DoseLog, error) {
		return scanDoseLog(row)
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&wakuwakuv1.ListDoseLogsResponse{DoseLogs: logs}), nil
}

const doseLogColumns = `id::text, medication_id::text, timing, scheduled_date, taken_at`

func findDoseLog(ctx context.Context, tx pgx.Tx, where string, args ...any) (*wakuwakuv1.DoseLog, error) {
	return scanDoseLog(tx.QueryRow(ctx, `SELECT `+doseLogColumns+` FROM dose_logs WHERE `+where, args...))
}

func scanDoseLog(row pgx.Row) (*wakuwakuv1.DoseLog, error) {
	var l wakuwakuv1.DoseLog
	var timing int16
	var date, takenAt time.Time
	if err := row.Scan(&l.Id, &l.MedicationId, &timing, &date, &takenAt); err != nil {
		return nil, err
	}
	l.Timing = wakuwakuv1.DoseTiming(timing)
	l.ScheduledDate = date.Format(dateLayout)
	l.TakenAt = timestamppb.New(takenAt)
	return &l, nil
}

// normalizeTimings は服用タイミングを検証し、重複を除いて昇順に並べる。
func normalizeTimings(in []wakuwakuv1.DoseTiming) ([]int16, error) {
	if len(in) == 0 {
		return nil, invalid("at least one timing is required")
	}
	out := make([]int16, 0, len(in))
	for _, t := range in {
		if t == wakuwakuv1.DoseTiming_DOSE_TIMING_UNSPECIFIED {
			return nil, invalid("timing must be specified")
		}
		if _, ok := wakuwakuv1.DoseTiming_name[int32(t)]; !ok {
			return nil, invalid(fmt.Sprintf("unknown timing %d", t))
		}
		out = append(out, int16(t))
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func toTimings(in []int16) []wakuwakuv1.DoseTiming {
	out := make([]wakuwakuv1.DoseTiming, len(in))
	for i, t := range in {
		out[i] = wakuwakuv1.DoseTiming(t)
	}
	return out
}

func requireUser(ctx context.Context) (string, error) {
	id, ok := auth.UserID(ctx)
	if !ok {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	return id, nil
}

func invalid(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func internal(err error) error {
	return connect.NewError(connect.CodeInternal, err)
}

func asConnectError(err error) error {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce
	}
	return internal(err)
}
