package watcher

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/config"
	"github.com/lcyvin/i3wm-hidden-workspaces/workspace-monitor/types"
	"go.i3wm.org/i3/v4"
)

type LayoutInstruction struct {
	Mark string `json:"mark"`
	Size Rect   `json:"dimensions"`
}

func GetLayoutInstructions(node *i3.Node, i3inst I3Instruction) []LayoutInstruction {
	inst := make([]LayoutInstruction, 0)
	for _, n := range node.Nodes {
		if len(n.Nodes) > 0 {
			inst = append(inst, GetLayoutInstructions(n, i3inst)...)
		} else {
			mark := i3inst.getMark(n.ID)
			if mark != "" {
				inst = append(inst, LayoutInstruction{
					Mark: mark,
					Size: Rect{n.Rect.Height, n.Rect.Width},
				})
			}
		}
	}
	return inst
}

type MarkID struct {
	ID     string           `json:"id"`
	UUID   string           `json:"uuid"`
	Layout types.LayoutType `json:"layout"`
}

type Rect struct {
	Height int64 `json:"height"`
	Width  int64 `json:"width"`
}

type I3Instruction struct {
	Data      string              `json:"-"`
	CMD       []types.I3Cmd       `json:"-"`
	Workspace string              `json:"-"`
	Marks     []MarkID            `json:"marks"`
	Layout    []LayoutInstruction `json:"layouts"`
}

func (i I3Instruction) runCmd(c types.I3Cmd) string {
	var fullCmd string
	switch c {
	case types.Hide:
		if len(i.Marks) > 0 {
			fullCmd = fmt.Sprintf(`[workspace="%s"] %s`, i.Workspace, c)
		}
	case types.Workspace:
		fullCmd = fmt.Sprintf(`%s %s`, c, i.Workspace)

	case types.Focus, types.Show:
		cmdProto := []string{}
		for _, m := range i.Marks {
			if m.UUID == "" {
				continue
			}
			cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] %s, floating disable`, m.UUID, c))
			cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] %s`, m.UUID, m.Layout))
		}
		fullCmd = strings.Join(cmdProto, "; ")

	case types.Mark:
		cmdProto := []string{}
		for _, m := range i.Marks {
			if m.UUID == "" {
				continue
			}
			cmdProto = append(cmdProto, fmt.Sprintf(`[con_id="%s"] %s %s`, m.ID, c, m.UUID))
		}
		fullCmd = strings.Join(cmdProto, "; ")

	case types.Load:
		cmdProto := []string{}
		for _, m := range i.Layout {
			cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] resize set width %d px height %d px`, m.Mark, m.Size.Width, m.Size.Height))
		}
		fullCmd = strings.Join(cmdProto, "; ")
	}

	return fullCmd
}

func (i I3Instruction) Run() types.Result {
	res := types.Result{}

	for _, cmd := range i.CMD {
		cmdString := i.runCmd(cmd)
		res.Msg = res.Msg + fmt.Sprintf("Running: %s\n", cmdString)
		_, err := i3.RunCommand(cmdString)
		if err != nil {
			res.Err = err
			return res
		}
	}

	return res
}

func (i I3Instruction) getMark(n i3.NodeID) string {
	for _, v := range i.Marks {
		if v.ID == fmt.Sprint(n) {
			return v.ID
		}
	}
	return ""
}

func GetMarkIDs(node *i3.Node) []MarkID {
	ids := make([]MarkID, 0)
	last := MarkID{}
	for i, n := range node.Nodes {
		if len(n.Nodes) > 0 {
			last.Layout = types.LayoutType(n.Layout)
			ids = append(ids, GetMarkIDs(n)...)
			ids[i].Layout = last.Layout
		} else {
			ids = append(ids, MarkID{
				ID:     fmt.Sprint(n.ID),
				UUID:   uuid.New().String(),
				Layout: types.LayoutType(n.Layout),
			})
		}
	}
	return ids
}

type I3Store struct {
	Workspace string
	Mark      string
}

func in(i string, col []string) bool {
	for _, v := range col {
		if i == v {
			return true
		}
	}
	return false
}

func Watcher(c config.Config, data chan<- I3Instruction) {
	recv := i3.Subscribe(i3.WorkspaceEventType)
	defer recv.Close()

	for recv.Next() {
		evt := recv.Event().(*i3.WorkspaceEvent)

		switch evt.Change {
		case "focus":
			if in(evt.Old.Name, c.Workspaces) {
				markIDs := GetMarkIDs(&evt.Old)
				cmd := I3Instruction{
					Data:      "store",
					CMD:       []types.I3Cmd{types.Mark, types.Save, types.Hide},
					Workspace: evt.Old.Name,
					Marks:     markIDs,
				}
				cmd.Layout = GetLayoutInstructions(&evt.Old, cmd)
				data <- cmd
			}

			if in(evt.Current.Name, c.Workspaces) {
				cmd := I3Instruction{
					Data:      "fetch",
					Workspace: evt.Current.Name,
					CMD:       []types.I3Cmd{types.Focus, types.Load},
				}
				data <- cmd
			}
		default:
			continue
		}
	}
}
