package util

import (
	"regexp"
	"testing"
)

func TestFindStringSubmatchMap(t *testing.T) {
	zk_pattern := `^(?P<time>[0-9,-: ]+) \[myid:(?P<id>\d+)\] - (?P<level>[A-Z]+) +\[(?P<tag>.+):(?P<class>[a-zA-Z_\$]+)@(?P<line>[0-9]+)\] - (?P<content>.+)$`
	zk_re := &MRegexp{regexp.MustCompile(zk_pattern)}
	zk_line := `2017-05-26 17:26:04,902 [myid:0] - INFO  [pano0/10.0.0.4:3888:QuorumCnxManager$Listener@511] - Received connection request /10.0.0.10:41628`
	result := zk_re.FindStringSubmatchMap(zk_line, "")
	if len(result) == 0 {
		t.Fatal("Expected to match line")
	}
	if result["time"] != "2017-05-26 17:26:04,902" {
		t.Fatal("Incorrect time field parsed")
	}
	if result["tag"] != "pano0/10.0.0.4:3888" {
		t.Fatal("Incorrect tag field parsed")
	}
}
