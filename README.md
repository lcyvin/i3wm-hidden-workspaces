# i3wm hidden workspaces
Does what it says on the tin. Define specific workspace names that should
"disappear" when they aren't focussed. Heavily abuses the scratchpad and ipc
features of i3wm.

## !!! THIS IS INCREDIBLY BUGGY, USE AT YOUR OWN RISK !!!
Seriously, calling this an "alpha build" is generous. The current codebase is
very hacky and does not fully implement the i3wm tree structure needed to
properly restore every possible type of workspace layout. At present, it is
capable of handling basic trees with splitv/splith windows and containers. Plans
for future additions of tabbed, stacking, and maybe floating windows are
tentative at best as I mostly wrote this to satisfy my own brain's desire to not
have a random workspace showing on my statusbar that is only ever used to hold
userspace background programs that I don't run via systemd.

*restoring layouts only partly working

# Why? 
To hide your daemons

# No but like, actually why?
Some applications, like mopidy/mpd, fan controllers, etc. need to run in
userland without being daemonized (or else are better suited to not being
daemonized) but aren't actively interacted with *that* frequently. I don't like
wasted space (which is part of why I use i3 to begin with), so I don't want a
random workspace using up space on my statusbar. Ipso facto, here's a tool that
lets you define a number of workspace names that you would like the daemon to
monitor. If they lose focus, their contents get marked and pushed into the
scratchpad void and hey presto, that workspace is no longer populated so it no
longer shows up in the statusbar. When you re-open the workspace (say, via
keybind), it fetches the appropriate windows and their layout and rebuilds the
workspace so you can do whatever it is you needed to do there.

# TODO
- make resizing windows actually work. I'm sure I've missed something in the i3
  docs, but at present resizing just fully does not work.
- implement better deamonization/architecture
  - Command line client with flag handlers and daemon spawning
  - Systemd unit installation for the local user
  - Config generation from cmd flags
  - Optional live config reloading
- Make/build files
- Tests
- Full support of i3's IPC featureset
  - Proper tree parsing and crawling
  - Layout restoration
    - [x] Window Splits
    - [x] Container Splits
    - [ ] Window Resizing
    - [ ] Container Resizing
    - [ ] Tabbed/Stacked Containers
    - [ ] Floating Windows

# How to use this tool

1. clone the repo: `git clone https://github.com/LcyVin/i3wm-hidden-workspaces`
2. build it `go build -o i3wm-hidden-workspaces workspace-monitor/main.go` (place this somewhere in your path)
3. Set a config file in `~/.config/i3wm-hidden-workspaces/config.yaml`. An example config is included. 
4. run and enjoy?

# Roadmap
- [ ] Fork core i3 communication abstractions to its own package
- [ ] Refactor monitor package to improve performance and logging support
- [ ] Add build files
- [ ] Command-line client
- [ ] Build automation
