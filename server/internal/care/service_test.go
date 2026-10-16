package care_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/investorkabuto/wakuwaku/server/internal/care"
	wakuwakuv1 "github.com/investorkabuto/wakuwaku/server/internal/gen/wakuwaku/v1"
	"github.com/investorkabuto/wakuwaku/server/internal/identity"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/auth"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/db"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	morning = wakuwakuv1.DoseTiming_DOSE_TIMING_MORNING
	evening = wakuwakuv1.DoseTiming_DOSE_TIMING_EVENING
	noon    = wakuwakuv1.DoseTiming_DOSE_TIMING_NOON
)

func setup(t *testing.T) (*care.Service, context.Context, context.Context) {
	t.Helper()
	pool := db.OpenForTest(t)
	ids := identity.NewStore(pool)
	ctx := context.Background()
	alice, err := ids.Resolve(ctx, "dev:alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := ids.Resolve(ctx, "dev:bob")
	if err != nil {
		t.Fatal(err)
	}
	return care.NewService(pool), auth.WithUserID(ctx, alice), auth.WithUserID(ctx, bob)
}

func createMed(t *testing.T, s *care.Service, ctx context.Context) *wakuwakuv1.Medication {
	t.Helper()
	res, err := s.CreateMedication(ctx, connect.NewRequest(&wakuwakuv1.CreateMedicationRequest{
		Name:     " 血圧のくすり ",
		DoseText: "1錠",
		Timings:  []wakuwakuv1.DoseTiming{evening, morning, morning},
	}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg.Medication
}

func record(s *care.Service, ctx context.Context, logID, medID string, timing wakuwakuv1.DoseTiming, confirm bool) (*wakuwakuv1.RecordDoseResponse, error) {
	res, err := s.RecordDose(ctx, connect.NewRequest(&wakuwakuv1.RecordDoseRequest{
		DoseLogId:        logID,
		MedicationId:     medID,
		Timing:           timing,
		ScheduledDate:    "2026-10-16",
		TakenAt:          timestamppb.New(time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)),
		ConfirmDuplicate: confirm,
	}))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

func code(err error) connect.Code {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return connect.CodeUnknown
}

func TestCreateMedicationNormalizes(t *testing.T) {
	s, alice, _ := setup(t)
	med := createMed(t, s, alice)
	if med.Name != "血圧のくすり" {
		t.Errorf("name = %q", med.Name)
	}
	if len(med.Timings) != 2 || med.Timings[0] != morning || med.Timings[1] != evening {
		t.Errorf("timings = %v", med.Timings)
	}
}

func TestCreateMedicationValidation(t *testing.T) {
	s, alice, _ := setup(t)
	cases := map[string]*wakuwakuv1.CreateMedicationRequest{
		"empty name":  {Name: " ", Timings: []wakuwakuv1.DoseTiming{morning}},
		"no timings":  {Name: "a"},
		"unspecified": {Name: "a", Timings: []wakuwakuv1.DoseTiming{0}},
		"unknown":     {Name: "a", Timings: []wakuwakuv1.DoseTiming{99}},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := s.CreateMedication(alice, connect.NewRequest(req))
			if code(err) != connect.CodeInvalidArgument {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestMedicationsAreIsolatedPerUser(t *testing.T) {
	s, alice, bob := setup(t)
	med := createMed(t, s, alice)

	res, err := s.ListMedications(bob, connect.NewRequest(&wakuwakuv1.ListMedicationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Msg.Medications) != 0 {
		t.Errorf("bob sees %d medications", len(res.Msg.Medications))
	}
	if _, err := record(s, bob, uuid.NewString(), med.Id, morning, false); code(err) != connect.CodeNotFound {
		t.Errorf("bob recording alice's medication: err = %v", err)
	}
}

func TestRecordDoseIsIdempotent(t *testing.T) {
	s, alice, _ := setup(t)
	med := createMed(t, s, alice)
	id := uuid.NewString()

	for i := range 2 {
		res, err := record(s, alice, id, med.Id, morning, false)
		if err != nil {
			t.Fatal(err)
		}
		if res.AlreadyRecorded || res.DoseLog.Id != id {
			t.Errorf("attempt %d: %+v", i, res)
		}
	}
	logs, err := s.ListDoseLogs(alice, connect.NewRequest(&wakuwakuv1.ListDoseLogsRequest{FromDate: "2026-10-16", ToDate: "2026-10-16"}))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(logs.Msg.DoseLogs); n != 1 {
		t.Errorf("got %d logs, want 1", n)
	}
}

func TestRecordDoseRequiresConfirmationForDuplicate(t *testing.T) {
	s, alice, _ := setup(t)
	med := createMed(t, s, alice)
	first, err := record(s, alice, uuid.NewString(), med.Id, morning, false)
	if err != nil {
		t.Fatal(err)
	}

	dup, err := record(s, alice, uuid.NewString(), med.Id, morning, false)
	if err != nil {
		t.Fatal(err)
	}
	if !dup.AlreadyRecorded || dup.DoseLog.Id != first.DoseLog.Id {
		t.Errorf("duplicate without confirm: %+v", dup)
	}

	confirmedID := uuid.NewString()
	confirmed, err := record(s, alice, confirmedID, med.Id, morning, true)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.AlreadyRecorded || confirmed.DoseLog.Id != confirmedID {
		t.Errorf("duplicate with confirm: %+v", confirmed)
	}

	// 別の回（夕）は確認なしで記録できる。
	if res, err := record(s, alice, uuid.NewString(), med.Id, evening, false); err != nil || res.AlreadyRecorded {
		t.Errorf("evening: res=%+v err=%v", res, err)
	}
}

func TestRecordDoseRejectsUnscheduledTiming(t *testing.T) {
	s, alice, _ := setup(t)
	med := createMed(t, s, alice)
	if _, err := record(s, alice, uuid.NewString(), med.Id, noon, false); code(err) != connect.CodeInvalidArgument {
		t.Errorf("err = %v", err)
	}
}

func TestRecordDoseRejectsReusedIDFromAnotherUser(t *testing.T) {
	s, alice, bob := setup(t)
	aliceMed := createMed(t, s, alice)
	bobMed := createMed(t, s, bob)
	id := uuid.NewString()
	if _, err := record(s, alice, id, aliceMed.Id, morning, false); err != nil {
		t.Fatal(err)
	}
	if _, err := record(s, bob, id, bobMed.Id, morning, false); code(err) != connect.CodeAlreadyExists {
		t.Errorf("err = %v", err)
	}
}
