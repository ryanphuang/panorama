package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type HealthServerConfig struct {
	Addr             string
	Id               string
	Subjects         []string
	Peers            map[string]string // all peers' id and address
	FilterSubmission bool              // whether to filter submitted report based on the subject id
	LogLevel         string
	DumpMemUsage     bool
	DBFile           string

	GCConfig  GarbageCollectionConfig
	BufConfig BufferingConfig
}

type GarbageCollectionConfig struct {
	Enable    bool
	Frequency int
	Threshold int
}

type BufferingConfig struct {
	HoldTime    int
	HoldListLen int
}

type ClassifierConfig struct {
	Context string
	Subject string
	Status  string
	Score   string
}

type FieldFilterClauseConfig struct {
	Field         string
	Operator      string
	Pattern       string
	CaptureResult bool // whether to capture filter result or just return the decision
}

type FieldFilterChainConfig struct {
	Chain      []*FieldFilterClauseConfig
	Classifier ClassifierConfig
}

type FieldFilterTreeConfig struct {
	FilterTree []*FieldFilterChainConfig
}

func LoadConfig(path string, config interface{}) error {
	fp, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	err = json.NewDecoder(fp).Decode(config)
	if err != nil {
		return err
	}
	return nil
}

func SaveConfig(path string, config interface{}) error {
	bytes, err := JSONMarshal(config, "", "    ")
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
	fmt.Printf("Config:\n%s", string(bytes))
	return fp.Close()
}

func JSONMarshal(t interface{}, prefix string, indent string) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(prefix, indent)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func JString(config interface{}) string {
	bytes, err := JSONMarshal(config, "", "    ")
	if err != nil {
		return ""
	}
	return string(bytes)
}
