package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReleaseGateEvaluate_Offline(t *testing.T) {
	svc := NewReleaseGateService(nil)
	r := svc.Evaluate()
	if !r.Overall {
		t.Fatalf("offline expected overall PASS, got %+v", r)
	}
	if len(r.Checks) == 0 {
		t.Fatal("expected checks")
	}
}

func TestReleaseGateEvaluate_WithDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM teacher_applications`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM teacher_payouts`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	svc := NewReleaseGateService(db)
	r := svc.Evaluate()
	if !r.Overall {
		t.Fatalf("expected PASS %+v", r)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
