package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"go.i3wm.org/i3/v4"

	badger "github.com/dgraph-io/badger/v3"
)

type i3Cmd string
type layoutType string

type Result struct {
	Err  error
	Msg  string
	Data interface{}
}

type KVEntry struct {
	Key   string
	Value interface{}
}

func New(key string, val []byte) *KVEntry {
	return &KVEntry{
		key,
		val,
	}
}

func NewFromStore(key string, db *badger.DB) (*KVEntry, error) {
	kve := &KVEntry{}
	kve.Key = key

	err := kve.Fetch(db)
	if err != nil {
		return kve, err
	}
	return kve, nil
}

func (kv *KVEntry) Store(db *badger.DB) error {
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(kv.Key), kv.Value.([]byte))
		return err
	})

	return err
}

func (kv *KVEntry) Fetch(db *badger.DB) error {
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(kv.Key))
		if err != nil {
			return err
		}

		item.Value(func(val []byte) error {
			var innerVal []byte
			copy(innerVal, val)
			kv.Value = innerVal
			return nil
		})
		return err
	})
	return err
}

// we'll do floating later absolutely fuck that for now
const (
	splith  layoutType = "splith"
	splitv             = "splitv"
	stacked            = "stacked"
	tabbed             = "tabbed"
)

const (
	mark      i3Cmd = "mark"
	hide            = "move scratchpad"
	show            = "scratchpad show"
	focus           = "focus"
	workspace       = "workspace"
	resize          = "resize"
	toggle          = "toggle"
	save            = "save" // this and load are meta-instructions for i3wm-hidden-workspaces
	load            = "load" // i3msg doesn't know what these mean in the context we need
)

type i3Instruction struct {
	Data      string
	CMD       []i3Cmd
	Workspace string
	Marks     []MarkID
	Layout    []layoutInstruction
}

type Rect struct {
	Height int64 `json:"height"`
	Width  int64 `json:"width"`
}

type layoutInstruction struct {
	Mark string
	Type layoutType `json:"type"`
	Size Rect       `json:"dimensions"`
}

type MarkID struct {
	ID   string `json:"id"`
	UUID string `json:"uuid"`
}

type i3Store struct {
	Workspace string
	Mark      string
}

func (i i3Instruction) runCmd(c i3Cmd) string {
	var fullCmd string
	switch c {
	case hide:
		fullCmd = fmt.Sprintf(`[workspace="%s"] %s`, i.Workspace, c)
	case workspace:
		fullCmd = fmt.Sprintf(`%s %s`, c, i.Workspace)
	case focus, show:
		cmd_proto := []string{}
		for _, m := range i.Marks {
			cmd_proto = append(cmd_proto, fmt.Sprintf(`[con_mark="%s"] %s, floating disable`, m.UUID, c))
		}
		fullCmd = strings.Join(cmd_proto, ";")
	case mark:
		cmd_proto := []string{}
		for _, m := range i.Marks {
			cmd_proto = append(cmd_proto, fmt.Sprintf(`[con_id="%s"] %s %s`, m.ID, c, m.UUID))
		}
		fullCmd = strings.Join(cmd_proto, ";")
	}

	return fullCmd
}

func (i i3Instruction) Run() error {
	for _, cmd := range i.CMD {
		fmt.Println("Running: " + i.runCmd(cmd))
		_, err := i3.RunCommand(i.runCmd(cmd))
		if err != nil {
			return err
		}
	}

	return nil
}

func getChildIDs(node *i3.Node) []string {
	ids := make([]string, 0)
	if len(node.Nodes) > 0 {
		for _, n := range node.Nodes {
			if len(n.Nodes) > 0 {
				ids = append(ids, getChildIDs(n)...)
			} else {
				ids = append(ids, fmt.Sprint(n.ID))
			}
		}
	}
	return ids
}

func getLayoutInstructions(node *i3.Node) []layoutInstruction {
	inst := make([]layoutInstruction, 0)

	// I do not have the brain power to do complex stacked/tabbed layouts at the moment
	if node.Layout == "tabbed" {
		inst = append(inst, layoutInstruction{Type: tabbed})
		return inst
	} else if node.Layout == "stacked" {
		inst = append(inst, layoutInstruction{Type: stacked})
		return inst
	}
	for _, n := range node.Nodes {
		if len(n.Nodes) > 0 {
			// recurse!
			inst = append(inst, getLayoutInstructions(n)...)
		} else {
			inst = append(inst, layoutInstruction{
				Type: layoutType(n.Layout),
				Mark: n.Marks[0],
				Size: Rect{n.Rect.Height, n.Rect.Width},
			})
		}
	}

	return inst
}

func focusMonitor(cmdData chan<- i3Instruction) {
	recv := i3.Subscribe(i3.WorkspaceEventType)
	defer recv.Close()

	for recv.Next() {
		evt := recv.Event().(*i3.WorkspaceEvent)

		switch evt.Change {
		case "focus":
			if evt.Old.Name == "test" {
				ids := getChildIDs(&evt.Old)
				marks := make([]MarkID, 0)

				for _, idx := range ids {
					marks = append(marks, MarkID{idx, uuid.New().String()})
				}
				cmd := i3Instruction{
					Data:      "store",
					CMD:       []i3Cmd{mark, save, hide},
					Workspace: evt.Old.Name,
					Marks:     marks,
					Layout:    getLayoutInstructions(&evt.Old),
				}
				cmdData <- cmd
			}

			if evt.Current.Name == "test" {
				cmd := i3Instruction{
					Data:      "fetch",
					Workspace: evt.Current.Name,
					CMD:       []i3Cmd{focus, load},
				}
				cmdData <- cmd
			}
		default:
		}
	}
}

func handleWatcherMsg(msg i3Instruction, db *badger.DB, result chan<- Result) {
	switch msg.Data {
	case "store":
		res := Result{}
		marks, _ := json.Marshal(msg.Marks)
		kve := New("workspace-"+msg.Workspace, marks)
		err := kve.Store(db)
		if err != nil {
			res.Err = err
			result <- res
			return
		}

		res.Msg = fmt.Sprintf("Stored key: workspace-%s, value: %s\n", msg.Workspace, marks)

		layout, _ := json.Marshal(msg.Layout)
		kve = New("layout-"+msg.Workspace, layout)
		err = kve.Store(db)
		if err != nil {
			res.Err = err
			result <- res
			return
		}

		res.Msg = res.Msg + fmt.Sprintf("Stored key: layout-%s, value: %s\n", msg.Workspace, layout)

		err = msg.Run()
		if err != nil {
			res.Err = err
			result <- res
			return
		}

	case "fetch":

	}
}

func main() {
	db, err := badger.Open(badger.DefaultOptions("/tmp/i3hider"))
	if err != nil {
		log.Fatalf("Unable to start database: %v", err)
	}
	defer db.Close()

	result := make(chan Result)
	cmdData := make(chan i3Instruction)
	go focusMonitor(cmdData)
	for {
		select {
		case msg := <-cmdData:
			go handleWatcherMsg(msg, db, result)
		case res := <-result:
			if res.Err != nil {
				fmt.Println(err)
			}
			if res.Msg != "" {
				fmt.Println(res.Msg)
			}
		}
	}
}
