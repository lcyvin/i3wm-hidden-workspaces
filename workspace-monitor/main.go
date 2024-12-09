package main

import (
	"encoding/json"
  "errors"
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
		res := types.Result{
      Msg: []string{},
    }
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
    resMsg := fmt.Sprintf("[STORE] KEY: %s, DATA: %s", msg.Workspace, marks)
    res.Msg = append(res.Msg, resMsg)

		runRes := msg.Run()
    if len(runRes.Msg) > 0 {
		  res.Msg = append(res.Msg, runRes.Msg...)
    }
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

    resMsg := fmt.Sprintf("[STORE] Attempting to fetch data for workspace: %s", msg.Workspace)
    res.Msg = append(res.Msg, resMsg)
		err := kve.Fetch(db)
		if err != nil {
			res.Err = errors.New(fmt.Sprintf("[STORE] %v", err))
			result <- res
		}

		if kve.Value != nil {
			err = json.Unmarshal(kve.Value, &msg)
			if err != nil {
				res.Err = errors.New(fmt.Sprintf("[JSON] %v", err))
				result <- res
			}

			runRes := msg.Run()
      res.Msg = append(res.Msg, runRes.Msg...)
			if runRes.Err != nil {
				res.Err = runRes.Err
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
			if len(res.Msg) > 0 {
        for _, msg := range res.Msg {
          fmt.Printf("INFO: %s\n", msg)
        }
			}
			if res.Err != nil {
        fmt.Printf("ERR: %v\n", res.Err)
			}
		}
	}
}
