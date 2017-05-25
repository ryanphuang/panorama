package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

var (
	config_str = `{"Chains":[{"Filters":[{"Field":"tag_context","Operator":"==","Pattern":"RecvWorker","CaptureResult":false}]},{"Filters":[{"Field":"tag_context","Operator":"==","Pattern":"SendWorker","CaptureResult":false}]},{"Filters":[{"Field":"tag_context","Operator":"~","Pattern":"WorkerReceiver\\[myid=(?P<rid>\\d)\\]","CaptureResult":true}]},{"Filters":[{"Field":"tag_context","Operator":"~","Pattern":"WorkerSender\\[myid=(?P<rid>\\d)\\]","CaptureResult":true}]},{"Filters":[{"Field":"tag_context","Operator":"~","Pattern":"LearnerHandler-/","CaptureResult":false},{"Field":"content","Operator":"~","Pattern":"^Slow serializing node .*$","CaptureResult":false}]},{"Filters":[{"Field":"tag_context","Operator":"==","Pattern":"Snapshot Thread","CaptureResult":false},{"Field":"content","Operator":"==","Pattern":"^Slow serializing node .*$","CaptureResult":false}]},{"Filters":[{"Field":"tag_context","Operator":"==","Pattern":"SyncThread","CaptureResult":false},{"Field":"content","Operator":"==","Pattern":"^Too busy to snap, skipping.*$","CaptureResult":false}]}]}`
)

func TestNewFieldFilterTree(t *testing.T) {
	config := new(FieldFilterPatternConfig)
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
	result["tag_context"] = "WorkerReceiver[myid=9]"
	ret, ok := tree.Eval(result)
	if !ok {
		t.Fatalf("Expected to match filter")
	}
	rid, ok := ret["tag_context_rid"]
	if !ok {
		t.Fatalf("Expected capture match result in `tag_context_rid`")
	}
	fmt.Printf("Captured tag_context_rid: %s\n", rid)
}
