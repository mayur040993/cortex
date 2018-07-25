package store

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/hashicorp/raft"
	"github.com/myntra/cortex/pkg/events"
	"github.com/myntra/cortex/pkg/executions"
	"github.com/myntra/cortex/pkg/rules"
)

type fsm defaultStore

func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch c.Op {
	case "stash":
		return f.applyStash(c.RuleID, c.Event)
	case "add_rule":
		return f.applyAddRule(c.Rule)
	case "update_rule":
		return f.applyUpdateRule(c.Rule)
	case "remove_rule":
		return f.applyRemoveRule(c.RuleID)
	case "flush_bucket":
		return f.applyFlushBucket(c.RuleID)
	case "add_script":
		return f.applyAddScript(c.ScriptID, c.Script)
	case "update_script":
		return f.applyUpdateScript(c.ScriptID, c.Script)
	case "remove_script":
		return f.applyRemoveScript(c.ScriptID)
	case "add_record":
		return f.applyAddRecord(c.Record)
	case "remove_record":
		return f.applyRemoveRecord(c.RecordID)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}

}

func (f *fsm) applyStash(ruleID string, event *events.Event) interface{} {
	return f.bucketStorage.stash(ruleID, event)
}

func (f *fsm) applyAddRule(rule *rules.Rule) interface{} {
	return f.bucketStorage.rs.addRule(rule)
}

func (f *fsm) applyUpdateRule(rule *rules.Rule) interface{} {
	return f.bucketStorage.rs.updateRule(rule)
}

func (f *fsm) applyRemoveRule(ruleID string) interface{} {
	return f.bucketStorage.rs.removeRule(ruleID)
}

func (f *fsm) applyFlushBucket(ruleID string) interface{} {
	return f.bucketStorage.es.flushBucket(ruleID)
}

func (f *fsm) applyAddScript(id string, script []byte) interface{} {
	return f.scriptStorage.addScript(id, script)
}

func (f *fsm) applyUpdateScript(id string, script []byte) interface{} {
	return f.scriptStorage.updateScript(id, script)
}

func (f *fsm) applyRemoveScript(id string) interface{} {
	return f.scriptStorage.removeScript(id)
}

func (f *fsm) applyAddRecord(r *executions.Record) interface{} {
	return f.executionStorage.add(r)
}

func (f *fsm) applyRemoveRecord(id string) interface{} {
	return f.executionStorage.remove(id)
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	glog.Info("snapshot <=")
	buckets := f.bucketStorage.es.clone()
	rules := f.bucketStorage.rs.clone()
	scripts := f.scriptStorage.clone()
	history := f.executionStorage.clone()
	return &fsmSnapShot{
		data: &DB{
			Buckets: buckets,
			Rules:   rules,
			Scripts: scripts,
			History: history,
		}}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	glog.Info("restore <=")
	defer rc.Close()
	var data DB

	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.bucketStorage.es.restore(data.Buckets)
	f.bucketStorage.rs.restore(data.Rules)
	f.scriptStorage.restore(data.Scripts)
	f.executionStorage.restore(data.History)

	return nil
}
