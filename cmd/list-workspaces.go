package main

import (
	"log"
  "fmt"

	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/config"
)

func main() {
  c, err := config.New()
  if err != nil {
    log.Fatal(err)
  }

  for _,v := range c.Workspaces {
    fmt.Println(v)
  }
}
