package store

import (
	"os"
	"strings"
	"sync"
	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	pb "panorama/build/gen"
	dt "panorama/types"
	du "panorama/util"
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

type HealthDBStorage struct {
	DB   *sql.DB
	File string

	insertReportStmt *sql.Stmt
	insertInferStmt  *sql.Stmt
	reportMu         *sync.Mutex
	inferMu          *sync.Mutex
}

func NewHealthDBStorage(file string) *HealthDBStorage {
	storage := &HealthDBStorage{
		File:     file,
		reportMu: &sync.Mutex{},
		inferMu:  &sync.Mutex{},
	}
	return storage
}

var _ dt.HealthDB = new(HealthDBStorage)

func (self *HealthDBStorage) Open() (*sql.DB, error) {
	if self.DB != nil {
		// if a db connection is already established
		// directly return that connection
		return self.DB, nil
	}
	if _, err := os.Stat(self.File); err == nil {
		du.LogI(sdtag, "Database %s already exists", self.File)
	}
	db, err := sql.Open("sqlite3", self.File)
	if err != nil {
		du.LogE(sdtag, "Fail to open database %s", self.File)
		return nil, err
	}
	_, err = db.Exec(CREATE_STMT)
	if err != nil {
		du.LogE(sdtag, "Fail to create database tables")
		db.Close()
		return nil, err
	}
	self.insertReportStmt, _ = db.Prepare(PANO_INSERT_STMT)
	self.insertInferStmt, _ = db.Prepare(INFER_INSERT_STMT)
	du.LogI(sdtag, "Database %s opened.", self.File)
	self.DB = db
	return db, nil
}

func (self *HealthDBStorage) InsertReport(report *pb.Report) error {
	if self.DB == nil {
		return nil
	}
	self.reportMu.Lock()
	defer self.reportMu.Unlock()

	ts := report.Observation.Ts
	lts := time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
	_, err := self.insertReportStmt.Exec(report.Subject, report.Observer, lts,
		dt.MetricsString(report.Observation.Metrics))
	if err != nil {
		du.LogE(sdtag, "Fail to insert report from %s to %s: %s", report.Observer, report.Subject, err)
	} else {
		du.LogD(sdtag, "Inserted report from %s to %s", report.Observer, report.Subject)
	}
	return err
}

func (self *HealthDBStorage) InsertInference(inf *pb.Inference) error {
	if self.DB == nil {
		return nil
	}
	self.inferMu.Lock()
	defer self.inferMu.Unlock()

	ts := inf.Observation.Ts
	lts := time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
	obs := strings.Join(inf.Observers, ",")
	_, err := self.insertInferStmt.Exec(inf.Subject, obs, lts, dt.MetricsString(inf.Observation.Metrics))
	if err != nil {
		du.LogE(sdtag, "Fail to insert inference from %s to %s: %s", obs, inf.Subject, err)
	} else {
		du.LogD(sdtag, "Inserted inference from %s to %s", obs, inf.Subject)
	}
	return err
}

func (self *HealthDBStorage) Close() {
	if self.DB != nil {
		self.DB.Close()
	}
}
