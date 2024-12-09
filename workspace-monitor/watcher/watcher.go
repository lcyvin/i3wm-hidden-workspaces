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
	ID              string            `json:"id"`
	UUID            string            `json:"uuid"`
	Layout          types.LayoutType  `json:"layout"`
  ParentNode      string            `json:"parent-node"`
  IsParent        bool              `json:"is-parent"`
  ContainerLayout types.LayoutType  `json:"container-layout"`
  Depth           int               `json:"depth"`
  Workspace       string            `json:"workspace"`
}

func NewMarkIDFromNode(ws string, node *i3.Node, d int, parent *MarkID) MarkID {
  m := MarkID{
    ID:     fmt.Sprint(node.ID),
    UUID:   uuid.New().String(),
    Depth:  d,
    Workspace: ws,
  }

  if parent != nil {
    m.ParentNode = parent.UUID
    if m.ParentNode != "" {
      m.ContainerLayout = parent.Layout
    }
  }

  if node.Layout != "" {
    m.Layout = types.LayoutType(node.Layout)
  }

  return m
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

func markFocus(mark MarkID) []string {
  cmdProto := make([]string, 0)
  // if the mark has a parent, focus that parent first so its container inheritence occurs
  if mark.ParentNode != "" {
    cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] focus`, mark.ParentNode))
  }

  // pop the window out of the scratchpad
  cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] scratchpad show, floating disable`, mark.UUID))

  // set the window itself's layout
  cmdProto = append(cmdProto, fmt.Sprintf(`[con_mark="%s"] %s`, mark.UUID, mark.Layout.Window()))

  return cmdProto
}

func markContainerize(mark MarkID) []string {
  return []string{fmt.Sprintf(`[con_mark="%s"] %s`, mark.UUID, mark.ContainerLayout.Container())}
}

func buildWorkspace(marks []MarkID) string {
  cmdProto := make([]string, 0)
  levels := 0
  for _,m := range marks {
    if m.UUID == "" {
      continue
    }

    if m.Depth > levels {
      levels = m.Depth
    }
  }

  // this is kindof a hack to get out of a corner I've coded myself into.
  // Starting to think about doing a V2...
  markTree := make([][]MarkID, 0)
  for loop := 0; loop <= levels; loop++ {
    markTree = append(markTree, []MarkID{})
    for _,m := range marks {
      if m.UUID == "" {
        continue
      }

      if m.Depth == loop {
        markTree[loop] = append(markTree[loop], m)
        //cmdProto = append(cmdProto, markFocusCommand(m)...)
      }
    }
  }

  // process the tree, first popping windows, then handling container layout
  for _, lvl := range markTree {
    for _, mark := range lvl {
      cmdProto = append(cmdProto, markFocus(mark)...)
    }
    for _, mark := range lvl {
      if mark.IsParent {
        cmdProto = append(cmdProto, markContainerize(mark)...)
      }
    }
  }

  fullCmd := strings.Join(cmdProto, "; ")

  return fullCmd
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
		fullCmd = buildWorkspace(i.Marks)

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
  res.Msg = []string{}

	for _, cmd := range i.CMD {
		cmdString := i.runCmd(cmd)
    if cmdString == "" {
      break
    }
		res.Msg = append(res.Msg, fmt.Sprintf("[RUNCMD] %s", cmdString))
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

func GetMarkIDs(ws string, node *i3.Node, depth int, seen map[i3.NodeID]MarkID, parent *MarkID) ([]MarkID, map[i3.NodeID]MarkID, bool) {
  if len(node.Nodes) == 0 && node.Type == "workspace" {
    return []MarkID{}, map[i3.NodeID]MarkID{}, true
  }

  ids := make([]MarkID, 0)
  crawlDepth := depth

	for _, n := range node.Nodes {
    if _,ok := seen[n.ID]; ok {
      continue
    }

		if len(n.Nodes) > 0 {
      containerLayout := n.Layout

      fncn := getFirstNonContainerNode(n)
      if fncn == nil {
        continue
      }
      fncnMark := NewMarkIDFromNode(ws, fncn, crawlDepth, parent)
      fncnMark.ContainerLayout = types.LayoutType(containerLayout)
      fncnMark.IsParent = true

      seen[fncn.ID] = fncnMark
      lastIdx := len(ids)-1
      if lastIdx < 0 {
        lastIdx = 0
      }
      ids = append(ids, fncnMark)
      
      gmi, nowSeen, _ := GetMarkIDs(ws, n, crawlDepth + 1, seen, &fncnMark)
      seen = nowSeen

      ids = append(ids, gmi...)

		} else {
      mid := NewMarkIDFromNode(ws, n, crawlDepth, parent)
      mid.IsParent = false
			ids = append(ids, mid)
    }
	}
	return ids, seen, false
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

func getFirstNonContainerNode(n *i3.Node) (*i3.Node) {
  if len(n.Nodes) == 0 && n.WindowProperties.Instance != "" {
    return n
  }

  for _,innerNode := range n.Nodes {
    if innerNode.WindowProperties.Instance != "" {
      return innerNode
    }

    res := getFirstNonContainerNode(innerNode)
    if res != nil {
      return res
    }
  }

  return nil
}

func Watcher(c config.Config, data chan<- I3Instruction) {
	recv := i3.Subscribe(i3.WorkspaceEventType)
	defer recv.Close()

	for recv.Next() {
		evt := recv.Event().(*i3.WorkspaceEvent)

		switch evt.Change {
		case "focus":
			if in(evt.Old.Name, c.Workspaces) {
				markIDs,_,_ := GetMarkIDs(evt.Old.Name, &evt.Old, 0, map[i3.NodeID]MarkID{}, nil)
				cmd := I3Instruction{
					Data:      "store",
					CMD:       []types.I3Cmd{types.Mark, types.Hide},
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
