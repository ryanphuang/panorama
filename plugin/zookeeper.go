package plugin

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	dt "deephealth/types"
	du "deephealth/util"
)

type ZooKeeperPlugin struct {
	Ensemble     []zkserver
	MyId         string
	FilterConfig *dt.FieldFilterTreeConfig
	Parser       dt.EventParser
}

type ZooKeeperEventParser struct {
	EntityIdPrefix string
	EIdAddrMap     map[string]string
	AddrEIdMap     map[string]string
	FilterTree     dt.FieldFilterTree
}

type zkserver struct {
	eid     string
	address string
}

type EventFilterConfig struct {
	TagContextPattern map[string]string
}

const (
	EID_PREFIX        = "peer@"
	CONF_ID_PREFIX    = "server."
	ZOOKEEPER_LINE_RE = `^(?P<time>[0-9,-: ]+) \[myid:(?P<id>\d+)\] - (?P<level>[A-Z]+) +\[(?P<tag>.+):(?P<class>[a-zA-Z_\$]+)@(?P<line>[0-9]+)\] - (?P<content>.+)$`
	TAG_ID_RE         = `^(?P<context>[a-zA-Z_\.\$]+):(?P<id>\d+)$`
	TAG_HOST_RE       = `^(?P<context>[a-zA-Z_\.\-\$]+):?(?P<source>[^/]*)/(?P<host>[^:]+):(?P<port>\d+)$`
)

var (
	ztag              = "zookeeper-plugin"
	zookeeperFlagset  = flag.NewFlagSet("zookeeper", flag.ExitOnError)
	zookeeperEnsemble = zookeeperFlagset.String("ensemble", "conf/zoo.cfg", "ZooKeeper ensemble file to use")
	zookeeperMyid     = zookeeperFlagset.String("myid", "conf/zoo_myid", "ZooKeeper myid file to use")
	zookeeperFilter   = zookeeperFlagset.String("filter", "conf/zoo_filter.json", "Filter configuration file to decide which event to report")
)

var (
	zkline_reg   = &du.MRegexp{regexp.MustCompile(ZOOKEEPER_LINE_RE)}
	tag_id_reg   = &du.MRegexp{regexp.MustCompile(TAG_ID_RE)}
	tag_host_reg = &du.MRegexp{regexp.MustCompile(TAG_HOST_RE)}
)

func NewZooKeeperEventParser(idprefix string, ensemble []zkserver, config *dt.FieldFilterTreeConfig) (*ZooKeeperEventParser, error) {
	m1 := make(map[string]string)
	m2 := make(map[string]string)
	for _, server := range ensemble {
		m1[server.eid] = server.address
		m2[server.address] = server.eid
	}
	m3, err := dt.NewFieldFilterTree(config)
	if err != nil {
		return nil, err
	}
	return &ZooKeeperEventParser{
		EntityIdPrefix: idprefix,
		EIdAddrMap:     m1,
		AddrEIdMap:     m2,
		FilterTree:     m3,
	}, nil
}

func (self *ZooKeeperEventParser) ParseLine(line string) *dt.Event {
	result := zkline_reg.FindStringSubmatchMap(line, "")
	if len(result) == 0 {
		return nil
	}
	if result["level"] == "DEBUG" {
		return nil
	}
	myid := result["id"]
	tag := result["tag"]
	content := result["content"]
	tag_result := tag_id_reg.FindStringSubmatchMap(tag, "")
	var tag_context string
	var tag_subject string
	var ok bool
	if len(tag_result) != 0 { // found potential EID in tag
		_, ok := self.EIdAddrMap[tag_result["id"]]
		if !ok {
			fmt.Fprintf(os.Stderr, "Tag id not in ensemble in log: %s\n", line)
			return nil
		}
		tag_subject = tag_result["id"] // EID in ensemble, assign it as tag subject
		tag_context = tag_result["context"]
	} else {
		tag_result = tag_host_reg.FindStringSubmatchMap(tag, "")
		// found potential host ip in tag
		if len(tag_result) != 0 && du.IsIP(tag_result["host"]) && du.IsPort(tag_result["port"]) {
			if tag_result["host"] == "0.0.0.0" {
				tag_subject = myid
			} else {
				tag_subject, ok = self.AddrEIdMap[tag_result["host"]]
				if !ok {
					fmt.Fprintf(os.Stderr, "Tag host not in ensemble in log: %s\n", line)
					return nil
				}
			}
			tag_context = tag_result["context"]
		} else {
			// a regular tag, to see if it is a self reporting tag
			// that might be interesting to others
			tag_subject = myid
			tag_context = tag
		}
	}
	result["tag_context"] = tag_context
	result["tag_subject"] = tag_subject
	ret, classifier, ok := self.FilterTree.Eval(result)
	if !ok {
		if tag_subject != myid {
			fmt.Fprintf(os.Stderr, "ignore communication related log: %s\n", line)
		}
		return nil
	}
	cres := classifier(ret)
	fmt.Println(tag_context, tag_subject, cres.Subject, cres.Status, cres.Score)
	var subject string
	var context string
	if len(cres.Subject) != 0 {
		subject, ok = self.AddrEIdMap[cres.Subject]
		if !ok {
			fmt.Fprintf(os.Stderr, "Filter host not in ensemble in log: %s\n", line)
			return nil
		}
	} else {
		subject = tag_subject
	}
	if len(cres.Context) != 0 {
		context = cres.Context
	} else {
		context = tag_context
	}
	if len(subject) == 0 {
		fmt.Fprintf(os.Stderr, "Empty subject in log: %s\n", line)
		return nil
	}
	timestamp, err := time.Parse("2006-01-02 15:04:05", result["time"][:19])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing timestamp %s: %s\n", result["time"], err)
		return nil
	}
	ms, err := strconv.Atoi(result["time"][20:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing timestamp %s: %s\n", result["time"], err)
		return nil
	}
	return &dt.Event{
		Time:    timestamp.Add(time.Millisecond * time.Duration(ms)),
		Id:      self.EntityIdPrefix + myid,
		Subject: self.EntityIdPrefix + subject,
		Context: context,
		Status:  cres.Status,
		Score:   cres.Score,
		Extra:   content,
	}
}

func ParseEnsembleFile(path string) ([]zkserver, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(fp)
	var ensemble []zkserver
	l := len(CONF_ID_PREFIX)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		idx := strings.IndexByte(line, '#')
		if idx >= 0 {
			line = line[:idx]
		}
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Ensemble file should have KEY=VALUE format")
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(key, CONF_ID_PREFIX) {
			continue
		}
		eid := key[l:]
		addr_str := strings.Split(value, ":")[0]
		ip := net.ParseIP(addr_str)
		if ip == nil {
			sips, err := net.LookupIP(addr_str)
			if err == nil {
				ensemble = append(ensemble, zkserver{eid: eid, address: sips[0].String()})
			} else {
				return nil, fmt.Errorf("Invalid address " + addr_str)
			}
		} else {
			ensemble = append(ensemble, zkserver{eid: eid, address: addr_str})
		}
	}
	if len(ensemble) == 0 {
		return nil, fmt.Errorf("No %sID=ADDRESS pair found", CONF_ID_PREFIX)
	}
	return ensemble, nil
}

func (self *ZooKeeperPlugin) ProvideFlags() *flag.FlagSet {
	return zookeeperFlagset
}

func (self *ZooKeeperPlugin) ValidateFlags() error {
	ensemble, err := ParseEnsembleFile(*zookeeperEnsemble)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(*zookeeperMyid)
	if err != nil {
		return err
	}
	myid := strings.TrimSpace(string(b))
	found := false
	for _, server := range ensemble {
		if myid == server.eid {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("My id %s is not in the ensemble", myid)
	}
	filterConfig := new(dt.FieldFilterTreeConfig)
	err = dt.LoadConfig(*zookeeperFilter, filterConfig)
	if err != nil {
		return err
	}
	fmt.Println(ensemble, filterConfig)
	self.Ensemble = ensemble
	self.FilterConfig = filterConfig
	return nil
}

func (self *ZooKeeperPlugin) Init() error {
	parser, err := NewZooKeeperEventParser(EID_PREFIX, self.Ensemble, self.FilterConfig)
	if err != nil {
		return err
	}
	self.Parser = parser
	return nil
}

func (self *ZooKeeperPlugin) ProvideEventParser() dt.EventParser {
	return self.Parser
}

func (self *ZooKeeperPlugin) ProvideObserverModule() dt.ObserverModule {
	return dt.ObserverModule{Module: "ZooKeeper", Observer: EID_PREFIX + self.MyId}
}
