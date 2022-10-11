# i3wm-hidden-workspaces

## !!! THIS IS INCREDIBLY BUGGY, USE AT YOUR OWN RISK !!!
## Seriously calling this an "alpha build" is generous

An incredibly hacky, not entirely working method of implementing numerous named scratchpads with restoring layouts*

*restoring layouts only partly working

# Why? 
To hide your daemons

# No but like, actually why?
Because I suck at C++ and this was easier than modifying polybar to filter out named workspaces while un-focussed

# TODO
- Sanity check marks so we don't write 80 million to a window when it only needs 1
- make resizing windows actually working
- implement client that can halt/pause the program's behavior

# How to use this tool

1. clone the repo: `git clone https://github.com/LcyVin/i3wm-hidden-workspaces`
2. build it `go build -o i3wm-hidden-workspaces workspace-monitor/main.go` (place this somewhere in your path)
3. Set a config file in `~/.config/i3wm-hidden-workspaces/config.yaml`. An example config is included. 
4. run and enjoy?
