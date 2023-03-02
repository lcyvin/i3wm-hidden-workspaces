package main

import (
	"encoding/json"
	"fmt"
	"log"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/config"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/kvs"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/types"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/watcher"
)

func handleWatcherMsg(msg watcher.I3Instruction, db *badger.DB, result chan<- types.Result) {
	switch msg.Data {
	case "store":
		res := types.Result{}
		marks, err := json.Marshal(msg)
		if err != nil {
			res.Err = err
			result <- res
		}

		kve := kvs.KVEntry{
			Key:   msg.Workspace,
			Value: marks,
		}

		err = kve.Store(db)
		if err != nil {
			res.Err = err
			result <- res
		}
		res.Msg = fmt.Sprintf("Stored key: %s, data: %s\n", msg.Workspace, marks)

		runRes := msg.Run()
		res.Msg = res.Msg + runRes.Msg
		if runRes.Err != nil {
			res.Err = runRes.Err
			result <- res
		}
		result <- res
	case "fetch":
		res := types.Result{}
		kve := kvs.KVEntry{
			Key: msg.Workspace,
		}

		fmt.Printf("Attempting to fetch data for workspace: %s\n", msg.Workspace)
		err := kve.Fetch(db)
		if err != nil {
			res.Err = err
			result <- res
		}

		if kve.Value != nil {
			err = json.Unmarshal(kve.Value, &msg)
			if err != nil {
				res.Err = err
				result <- res
			}

			runRes := msg.Run()
			if runRes.Err != nil {
				res.Err = runRes.Err
				res.Msg = res.Msg + runRes.Msg
				result <- res
			}

			result <- res
		}
	}
}

func main() {
	c, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := kvs.New(c.DBFile, c.PersistentDB)
	if err != nil {
		log.Fatal(err)
	}

	result := make(chan types.Result)
	data := make(chan watcher.I3Instruction)

	defer db.Close()

	go watcher.Watcher(c, data)
	for {
		select {
		case inc := <-data:
			go handleWatcherMsg(inc, db, result)
		case res := <-result:
			if res.Msg != "" {
				fmt.Println(res.Msg)
			}
			if res.Err != nil {
				fmt.Println(res.Err)
			}
		}
	}
}
