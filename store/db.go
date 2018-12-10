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
		CREATE TABLE IF NOT EXISTS registration (id INTEGER PRIMARY KEY, handle INTEGER, module TEXT, observer TEXT, time TIMESTAMP);
	`
	PANO_INSERT_STMT     = "INSERT INTO panorama(subject, observer, time, metrics) VALUES(?,?,?,?)"
	INFER_INSERT_STMT    = "INSERT INTO inference(subject, observers, time, metrics) VALUES(?,?,?,?)"
	REGISTER_INSERT_STMT = "INSERT INTO registration(handle, module, observer, time) VALUES(?,?,?,?)"
)

type HealthDBStorage struct {
	DB   *sql.DB
	File string

	insertReportStmt   *sql.Stmt
	insertInferStmt    *sql.Stmt
	insertRegisterStmt *sql.Stmt
	reportMu           *sync.Mutex
	inferMu            *sync.Mutex
	regMu              *sync.Mutex
}

func NewHealthDBStorage(file string) *HealthDBStorage {
	storage := &HealthDBStorage{
		File:     file,
		reportMu: &sync.Mutex{},
		inferMu:  &sync.Mutex{},
		regMu:    &sync.Mutex{},
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
	self.insertRegisterStmt, _ = db.Prepare(REGISTER_INSERT_STMT)
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

func (self *HealthDBStorage) InsertRegistration(reg *dt.Registration) error {
	if self.DB == nil {
		return nil
	}
	self.regMu.Lock()
	defer self.regMu.Unlock()
	du.LogI(sdtag, "Inserting registration %v", reg)
	_, err := self.insertRegisterStmt.Exec(reg.Handle, reg.Module, reg.Observer, reg.Time)
	if err != nil {
		du.LogE(sdtag, "Fail to insert registration from %s: %s", reg.Observer, err)
	} else {
		du.LogD(sdtag, "Inserted registration from %s", reg.Observer)
	}
	return err
}

func (self *HealthDBStorage) ReadRegistrations() (map[uint64]*dt.Registration, uint64) {
	if self.DB == nil {
		return nil, 0
	}
	du.LogI(sdtag, "Reading previous registrations...")
	rows, err := self.DB.Query("SELECT handle, module, observer, time FROM registration")
	if err != nil {
		du.LogE(sdtag, "Fail to read registrations %s", err)
		return nil, 0
	}
	defer rows.Close()
	registrations := make(map[uint64]*dt.Registration)
	var max_handle uint64 = 0
	for rows.Next() {
		var handle uint64
		var module string
		var observer string
		var ts time.Time
		err = rows.Scan(&handle, &module, &observer, &ts)
		if err == nil {
			reg, ok := registrations[handle]
			if ok {
				if reg.Time.Before(ts) {
					observer := dt.ObserverModule{Module: module, Observer: observer}
					newreg := &dt.Registration{ObserverModule: observer, Handle: handle, Time: ts}
					du.LogI(sdtag, "Overwrite an existing registration %v with %v", reg, newreg)
					registrations[handle] = newreg
				}
			} else {
				observer := dt.ObserverModule{Module: module, Observer: observer}
				newreg := &dt.Registration{ObserverModule: observer, Handle: handle, Time: ts}
				registrations[handle] = newreg
				du.LogI(sdtag, "Read an existing registration %v", newreg)
			}
			if handle > max_handle {
				max_handle = handle
			}
		} else {
			du.LogE(sdtag, "Failed to read registration: %s", err)
		}
	}
	du.LogI(sdtag, "Done reading previous registrations")
	return registrations, max_handle
}

func (self *HealthDBStorage) Close() {
	if self.DB != nil {
		self.DB.Close()
	}
}
