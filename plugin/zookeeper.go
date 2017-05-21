package plugin

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	dt "deephealth/types"
	du "deephealth/util"
)

type ZooKeeperPlugin struct {
	Ensemble []zkserver
	Parser   dt.EventParser
}

type ZooKeeperEventParser struct {
	EntityIdPrefix  string
	EIdAddrMap      map[string]string
	AddrEIdMap      map[string]string
	TagContextReMap du.MRegexpMap
}

type zkserver struct {
	eid     string
	address string
}

const (
	EID_PREFIX        = "peer@"
	CONF_ID_PREFIX    = "server."
	ZOOKEEPER_LINE_RE = `^(?P<time>[0-9,-: ]+) \[myid:(?P<id>\d+)\] - (?P<level>[A-Z]+) +\[(?P<tag>.+):(?P<class>[a-zA-Z_\$]+)@(?P<line>[0-9]+)\] - (?P<content>.+)$`
	TAG_ID_RE         = `^(?P<context>[a-zA-Z_\.\$]+):(?P<id>\d+)$`
	TAG_HOST_RE       = `^(?P<context>[a-zA-Z_\.\-\$]+):?(?P<source>[^/]*)/(?P<host>[^:]+):(?P<port>\d+)$`
)

var (
	zookeeperFlagset  = flag.NewFlagSet("zookeeper", flag.ExitOnError)
	zookeeperEnsemble = zookeeperFlagset.String("ensemble", "zoo.cfg", "ZooKeeper ensemble file to use")
)

var (
	zkline_reg           = &du.MRegexp{regexp.MustCompile(ZOOKEEPER_LINE_RE)}
	tag_id_reg           = &du.MRegexp{regexp.MustCompile(TAG_ID_RE)}
	tag_host_reg         = &du.MRegexp{regexp.MustCompile(TAG_HOST_RE)}
	tag_context_patterns = map[string]string{
		"RecvWorker":       "",
		"SendWorker":       "",
		"SyncThread":       "^Too busy to snap, skipping.*$",
		"Snapshot Thread":  "^Slow serializing node .*$",
		"LearnerHandler-/": "^Slow serializing node .*$",
	}
)

func NewZooKeeperEventParser(idprefix string, ensemble []zkserver, tag_context_patterns map[string]string) *ZooKeeperEventParser {
	m1 := make(map[string]string)
	m2 := make(map[string]string)
	for _, server := range ensemble {
		m1[server.eid] = server.address
		m2[server.address] = server.eid
	}
	m3 := du.NewMRegexpMap(tag_context_patterns)
	return &ZooKeeperEventParser{
		EntityIdPrefix:  idprefix,
		EIdAddrMap:      m1,
		AddrEIdMap:      m2,
		TagContextReMap: m3,
	}
}

func (self *ZooKeeperEventParser) ParseLine(line string) *dt.Event {
	result := zkline_reg.FindStringSubmatchMap(line)
	if len(result) == 0 {
		return nil
	}
	if result["level"] == "INFO" || result["level"] == "DEBUG" {
		return nil
	}
	myid := result["id"]
	tag := result["tag"]
	content := result["content"]
	tag_result := tag_id_reg.FindStringSubmatchMap(tag)
	var tag_context string
	var tag_subject string
	var ok bool
	if len(tag_result) != 0 { // found potential EID in tag
		_, ok := self.EIdAddrMap[tag_result["id"]]
		if !ok {
			return nil
		}
		tag_subject = tag_result["id"] // EID in ensemble, assign it as tag subject
		tag_context = tag_result["context"]
	} else {
		tag_result = tag_host_reg.FindStringSubmatchMap(tag)
		// found potential host ip in tag
		if len(tag_result) != 0 && du.IsIP(tag_result["host"]) && du.IsPort(tag_result["port"]) {
			if tag_result["host"] == "0.0.0.0" {
				tag_subject = myid
			} else {
				tag_subject, ok = self.AddrEIdMap[tag_result["host"]]
				if !ok {
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
	re, ok := self.TagContextReMap[tag_context]
	if !ok || (re != nil && !re.MatchString(content)) {
		return nil
	}
	if len(tag_subject) == 0 {
		return nil
	}
	timestamp, err := time.Parse("2006-01-02 15:04:05", result["time"][:19])
	if err != nil {
		return nil
	}
	return &dt.Event{
		Time:    timestamp,
		Id:      self.EntityIdPrefix + myid,
		Subject: self.EntityIdPrefix + tag_subject,
		Context: tag_context,
		Extra:   content,
	}
}

func (self *ZooKeeperPlugin) ProvideFlags() *flag.FlagSet {
	return zookeeperFlagset
}

func (self *ZooKeeperPlugin) ValidateFlags() error {
	fp, err := os.Open(*zookeeperEnsemble)
	if err != nil {
		return err
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
			return fmt.Errorf("Ensemble file should have KEY=VALUE format")
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
				return fmt.Errorf("Invalid address " + addr_str)
			}
		} else {
			ensemble = append(ensemble, zkserver{eid: eid, address: addr_str})
		}
	}
	if len(ensemble) == 0 {
		return fmt.Errorf("No %sID=ADDRESS pair found", CONF_ID_PREFIX)
	}
	self.Ensemble = ensemble
	fmt.Println(ensemble)
	return nil
}

func (self *ZooKeeperPlugin) Init() error {
	self.Parser = NewZooKeeperEventParser(EID_PREFIX, self.Ensemble, tag_context_patterns)
	return nil
}

func (self *ZooKeeperPlugin) ProvideEventParser() dt.EventParser {
	return self.Parser
}
