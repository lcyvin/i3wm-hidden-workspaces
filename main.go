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

const (
	mark      i3Cmd = "mark"
	hide            = "move scratchpad"
	show            = "scratchpad show"
	focus           = "focus"
	workspace       = "workspace"
)

type i3Instruction struct {
	Data      string
	CMD       []i3Cmd
	Workspace string
	Marks     []MarkID
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
					CMD:       []i3Cmd{mark, hide},
					Workspace: evt.Old.Name,
					Marks:     marks,
				}
				cmdData <- cmd
			}

			if evt.Current.Name == "test" {
				cmd := i3Instruction{
					Data:      "fetch",
					Workspace: evt.Current.Name,
					CMD:       []i3Cmd{focus},
				}
				cmdData <- cmd
			}
		default:
		}
	}
}

func main() {
	db, err := badger.Open(badger.DefaultOptions("/tmp/i3hider"))
	if err != nil {
		log.Fatalf("Unable to start database: %v", err)
	}
	defer db.Close()

	cmdData := make(chan i3Instruction)
	go focusMonitor(cmdData)
	for {
		select {
		case msg := <-cmdData:
			if msg.Data == "store" {
				marks, _ := json.Marshal(msg.Marks)
				err := db.Update(func(txn *badger.Txn) error {
					err := txn.Set([]byte(msg.Workspace), marks)
					return err
				})
				fmt.Println(err)
				err = msg.Run()
				if err != nil {
					fmt.Println(err)
				}
			} else if msg.Data == "fetch" {
				err := db.View(func(txn *badger.Txn) error {
					item, err := txn.Get([]byte(msg.Workspace))
					if err != nil {
						fmt.Println(err)
						return err
					}

					item.Value(func(val []byte) error {
						marks := make([]MarkID, 0)
						json.Unmarshal(val, &marks)
						msg.Marks = marks
						return nil
					})
					return nil
				})
				if err != nil {
					fmt.Println(err)
				}
				err = msg.Run()
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}
