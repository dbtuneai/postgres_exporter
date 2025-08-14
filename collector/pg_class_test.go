package collector

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
)

func TestPGClassCollector(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	inst := &instance{db: db}

	columns := []string{
		"datname", "schemaname", "relname", "relfrozenxid", "relfrozenxid_age",
		"relminmxid", "relminmxid_age", "relpages", "reltuples", "relnatts",
		"relallvisible",
	}
	rows := sqlmock.NewRows(columns).
		AddRow("postgres", "public", "table", 1000, 1, 2000, 2, 5, 100, 17, 3)
	mock.ExpectQuery(sanitizeQuery(pgClassQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		c := PGClassCollector{}

		if err := c.Update(context.Background(), inst, ch); err != nil {
			t.Errorf("Error calling PGStatStatementsCollector.Update: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_COUNTER, value: 1000},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 1},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_COUNTER, value: 2000},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 2},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 5},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 100},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 17},
		{labels: labelMap{"datname": "postgres", "schemaname": "public", "relname": "table"}, metricType: dto.MetricType_GAUGE, value: 3},
	}

	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range expected {
			m := readMetric(<-ch)
			convey.So(expect, convey.ShouldResemble, m)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
