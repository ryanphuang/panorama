package types

import (
	"encoding/json"
	"fmt"
	"os"
)

type RC struct {
	HealthServers map[EntityId]string
}

func LoadRC(path string) (*RC, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	rc := new(RC)
	err = json.NewDecoder(fp).Decode(rc)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (self *RC) Save(path string) error {
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

func (self *RC) marshal() ([]byte, error) {
	return json.MarshalIndent(self, "", "    ")
}

func (self *RC) String() string {
	bytes, err := self.marshal()
	if err != nil {
		return ""
	}
	return string(bytes)
}
