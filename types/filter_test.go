package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

var (
	config_str = `{"FilterTree":[{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"RecvWorker","CaptureResult":false}],"Classifier":{"Context":"","Subject":"","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"SendWorker","CaptureResult":false}],"Classifier":{"Context":"","Subject":"","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"tag_context","Operator":"~","Pattern":"WorkerSender\\[myid=\\d+\\]","CaptureResult":false},{"Field":"content","Operator":"~","Pattern":"Cannot open channel to (?P<rid>\\d+) at election address .*/(?P<host>[^:]+):(?P<port>\\d+)","CaptureResult":true}],"Classifier":{"Context":"WorkerSender","Subject":"<content_host>","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"LearnerHandler-","CaptureResult":false},{"Field":"content","Operator":"(~","Pattern":"['^Slow serializing node .*$', '^Unexpected exception causing shutdown .*$', '.* GOODBYE .*$']","CaptureResult":false}],"Classifier":{"Context":"LearnerHandler","Subject":"","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"LearnerHandler-","CaptureResult":false},{"Field":"content","Operator":"(~","Pattern":"['^Synchronizing with Follower sid: .*$', '^Received NEWLEADER-ACK message .*$']","CaptureResult":false}],"Classifier":{"Context":"LearnerHandler","Subject":"","Status":"healthy","Score":"90"}},{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"Snapshot Thread","CaptureResult":false},{"Field":"content","Operator":"~","Pattern":"^Slow serializing node .*$","CaptureResult":false}],"Classifier":{"Context":"Snapshot","Subject":"","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"tag_context","Operator":"==","Pattern":"SyncThread","CaptureResult":false},{"Field":"content","Operator":"~","Pattern":"^Too busy to snap, skipping.*$","CaptureResult":false}],"Classifier":{"Context":"","Subject":"","Status":"unhealthy","Score":"20"}},{"Chain":[{"Field":"class","Operator":"==","Pattern":"QuorumCnxManager$Listener","CaptureResult":false},{"Field":"content","Operator":"~","Pattern":"Received connection request /(?P<host>[^:]+):(?P<port>\\d+)","CaptureResult":true}],"Classifier":{"Context":"QuorumListener","Subject":"<content_host>","Status":"healthy","Score":"90"}}]}`
)

func TestNewFieldFilterTree(t *testing.T) {
	config := new(FieldFilterTreeConfig)
	err := json.Unmarshal([]byte(config_str), config)
	if err != nil {
		t.Fatal("Fail to parse config string: %s\n", err)
	}
	// fmt.Println(JString(config))
	tree, err := NewFieldFilterTree(config)
	if err != nil {
		t.Fatalf("Fail to create filter tree from config: %s\n", err)
	}
	result := make(map[string]string)
	result["tag_context"] = "WorkerSender[myid=9]"
	result["content"] = "Cannot open channel to 1 at election address pano1/10.0.0.5:3888"
	ret, _, ok := tree.Eval(result)
	if !ok {
		t.Fatalf("Expected to match filter")
	}
	rid, ok := ret["content_rid"]
	if !ok {
		t.Fatalf("Expected capture match result in `tag_context_rid`")
	}
	fmt.Printf("Captured content_rid: %s\n", rid)
}
