package types

type I3Cmd string
type LayoutType string

func (lt LayoutType) Window() string {
  switch lt {
  case Splith:
    return "splith"
  case Splitv:
    return "splitv"
  }

  return ""
}

func (lt LayoutType) Container() string {
  switch lt {
  case Splith:
    return "split horizontal"
  case Splitv:
    return "split vertical"
  }

  return ""
}

const (
	Splith  LayoutType = "splith"
	Splitv             = "splitv"
	Stacked            = "stacked"
	Tabbed             = "tabbed"
)

const (
	Mark      I3Cmd = "mark"
	Hide            = "move scratchpad"
	Show            = "scratchpad show"
	Focus           = "focus"
	Workspace       = "workspace"
	Resize          = "resize"
	Toggle          = "toggle"
	Save            = "save" // this and load are meta-instructions for i3wm-hidden-workspaces
	Load            = "load" // i3msg doesn't know what these mean in the context we need
)

type Result struct {
	Err  error
	Msg  []string
	Data interface{}
}
