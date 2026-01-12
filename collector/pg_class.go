// The definition of the metrics here were inspired on another pg_exporter
// implementation below:
// https://github.com/pgsty/pg_exporter/blob/d42965d98c328b6ef203e3322895d70bdebd7d14/config/0700-pg_table.yml#L16
// The selected metrics are based on this issue with some additions
// https://github.com/prometheus-community/postgres_exporter/issues/260

package collector

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

const classSubsystem = "class"

func init() {
	registerCollector(classSubsystem, defaultDisabled, NewPGClassCollector)
}

type PGClassCollector struct {
	log *slog.Logger
}

func NewPGClassCollector(config collectorConfig) (Collector, error) {
	return &PGClassCollector{
		log: config.logger,
	}, nil
}

var (
	classRelFrozenXid = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relfrozenxid"),
		"All txids before this have been frozen on the table",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelFrozenXidAge = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relfrozenxid_age"),
		"The age of this table in vacuum cycles computed as age(relfrozenxid)",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelMinMxid = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relminmxid"),
		"All mxids before this one have been replaced by a transaction ID on the table",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelMinMxidAge = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relminmxid_age"),
		"The age of this table in vacuum cycles computed as age(relminmxid)",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelPages = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relpages"),
		"Size of the on-disk representation of this table in pages",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelTuples = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "reltuples"),
		"Number of live rows in the table. This is only an estimate used by the planner.",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelNatts = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relnatts"),
		"Number of user columns in the relation",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)

	classRelAllVisible = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, classSubsystem, "relallvisible"),
		"Number of pages that are marked all-visible in the table's visibility map. This is only an estimate used by the planner.",
		[]string{"datname", "schemaname", "relname"},
		prometheus.Labels{},
	)
)

const (
	// Top 100 biggest tables
	// This limit is based on the pg_stat_statements but can be changed.
	pgClassQuery = `SELECT
		current_database() AS datname,
		stat.schemaname,
		stat.relname,
		cls.relfrozenxid,
		age(cls.relfrozenxid) AS relfrozenxid_age,
		cls.relminmxid,
		mxid_age(cls.relminmxid) AS relminmxid_age,
		cls.relpages,
		cls.reltuples,
		cls.relnatts,
		cls.relallvisible
	FROM
		pg_stat_user_tables AS stat
	LEFT JOIN
		pg_class AS cls ON stat.relid = cls.oid
	ORDER BY cls.relpages DESC
	LIMIT 100;`
)

func (c PGClassCollector) Update(ctx context.Context, instance *instance, ch chan<- prometheus.Metric) error {
	var query = pgClassQuery

	db := instance.getDB()
	rows, err := db.QueryContext(ctx, query)

	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var datname, schemaname, relname sql.NullString
		var relfrozenxid, relfrozenxidAge, relminmxid, relminmxidAge, relpages, relallvisible sql.NullInt32
		var reltuples sql.NullFloat64
		var relnatts sql.NullInt16

		if err := rows.Scan(&datname, &schemaname, &relname, &relfrozenxid, &relfrozenxidAge, &relminmxid, &relminmxidAge, &relpages, &reltuples, &relnatts, &relallvisible); err != nil {
			return err
		}

		datnameLabel := "unknown"
		if datname.Valid {
			datnameLabel = datname.String
		}
		schemanameLabel := "unknown"
		if schemaname.Valid {
			schemanameLabel = schemaname.String
		}
		relnameLabel := "unknown"
		if relname.Valid {
			relnameLabel = relname.String
		}

		relfrozenxidMetric := 0.0
		if relfrozenxid.Valid {
			relfrozenxidMetric = float64(relfrozenxid.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelFrozenXid,
			prometheus.CounterValue,
			relfrozenxidMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relfrozenxidAgeMetric := 0.0
		if relfrozenxidAge.Valid {
			relfrozenxidAgeMetric = float64(relfrozenxidAge.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelFrozenXidAge,
			prometheus.GaugeValue,
			relfrozenxidAgeMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relminmxidMetric := 0.0
		if relminmxid.Valid {
			relminmxidMetric = float64(relminmxid.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelMinMxid,
			prometheus.CounterValue,
			relminmxidMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relminmxidAgeMetric := 0.0
		if relminmxidAge.Valid {
			relminmxidAgeMetric = float64(relminmxidAge.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelMinMxidAge,
			prometheus.GaugeValue,
			relminmxidAgeMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relpagesMetric := 0.0
		if relpages.Valid {
			relpagesMetric = float64(relpages.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelPages,
			prometheus.GaugeValue,
			relpagesMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		reltuplesMetric := 0.0
		if reltuples.Valid {
			reltuplesMetric = float64(reltuples.Float64)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelTuples,
			prometheus.GaugeValue,
			reltuplesMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relnattsMetric := 0.0
		if relnatts.Valid {
			relnattsMetric = float64(relnatts.Int16)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelNatts,
			prometheus.GaugeValue,
			relnattsMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)

		relallvisibleMetric := 0.0
		if relallvisible.Valid {
			relallvisibleMetric = float64(relallvisible.Int32)
		}
		ch <- prometheus.MustNewConstMetric(
			classRelAllVisible,
			prometheus.GaugeValue,
			relallvisibleMetric,
			datnameLabel, schemanameLabel, relnameLabel,
		)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}
