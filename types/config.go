package types

import (
	"encoding/json"
	"fmt"
	"os"
)

type HealthServerConfig struct {
	Addr             string
	Id               EntityId
	Subjects         []EntityId
	Peers            map[EntityId]string // all peers' id and address
	FilterSubmission bool                // whether to filter submitted report based on the subject id
}

func LoadConfig(path string) (*HealthServerConfig, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	rc := new(HealthServerConfig)
	err = json.NewDecoder(fp).Decode(rc)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (self *HealthServerConfig) Save(path string) error {
	bytes, err := self.marshal()
	if err != nil {
		return err
	}
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = fp.Write(bytes)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(fp)
	if err != nil {
		return err
	}
	return fp.Close()
}

func (self *HealthServerConfig) marshal() ([]byte, error) {
	return json.MarshalIndent(self, "", "    ")
}

func (self *HealthServerConfig) String() string {
	bytes, err := self.marshal()
	if err != nil {
		return ""
	}
	return string(bytes)
}
