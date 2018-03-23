package store

import (
	"os"
	"strings"
	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	pb "deephealth/build/gen"
	dt "deephealth/types"
	du "deephealth/util"
)

const (
	sdtag       = "db"
	DB_FILE     = "deephealth.db"
	CREATE_STMT = `
		CREATE TABLE IF NOT EXISTS panorama (id INTEGER PRIMARY KEY, subject TEXT, observer TEXT, time TIMESTAMP, metrics TEXT);
		CREATE TABLE IF NOT EXISTS inference (id INTEGER PRIMARY KEY, subject TEXT, observers TEXT, time TIMESTAMP, metrics TEXT);
	`
	PANO_INSERT_STMT  = "INSERT INTO panorama(subject, observer, time, metrics) VALUES(?,?,?,?)"
	INFER_INSERT_STMT = "INSERT INTO inference(subject, observers, time, metrics) VALUES(?,?,?,?)"
)

var insertPanoStmt *sql.Stmt
var insertInferStmt *sql.Stmt

func CreateDB() (*sql.DB, error) {
	if _, err := os.Stat(DB_FILE); err == nil {
		du.LogI(sdtag, "Database %s already exists", DB_FILE)
	}
	db, err := sql.Open("sqlite3", DB_FILE)
	if err != nil {
		du.LogE(sdtag, "Fail to open database %s", DB_FILE)
		return nil, err
	}
	_, err = db.Exec(CREATE_STMT)
	if err != nil {
		du.LogE(sdtag, "Fail to create database tables")
		db.Close()
		return nil, err
	}
	insertPanoStmt, _ = db.Prepare(PANO_INSERT_STMT)
	insertInferStmt, _ = db.Prepare(INFER_INSERT_STMT)
	return db, nil
}

func InsertReport(db *sql.DB, report *pb.Report) error {
	if db == nil {
		return nil
	}
	ts := report.Observation.Ts
	lts := time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
	_, err := insertPanoStmt.Exec(report.Subject, report.Observer, lts, dt.MetricsString(report.Observation.Metrics))
	if err != nil {
		du.LogE(sdtag, "Fail to insert report from %s to %s: %s", report.Observer, report.Subject, err)
	} else {
		du.LogI(sdtag, "Inserted report from %s to %s", report.Observer, report.Subject)
	}
	return err
}

func InsertInference(db *sql.DB, inf *pb.Inference) error {
	if db == nil {
		return nil
	}
	ts := inf.Observation.Ts
	lts := time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
	obs := strings.Join(inf.Observers, ",")
	_, err := insertInferStmt.Exec(inf.Subject, obs, lts, dt.MetricsString(inf.Observation.Metrics))
	if err != nil {
		du.LogE(sdtag, "Fail to insert inference from %s to %s: %s", obs, inf.Subject, err)
	} else {
		du.LogI(sdtag, "Inserted inference from %s to %s", obs, inf.Subject)
	}
	return err
}
