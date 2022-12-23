# Ziggs

  ![Ziggs Demo](https://tcp.ac/i/wW3Fh.gif)

There are zigbees in my head.

### Huh?

This tool was only planned as a personal tool for myself, consequently the codebase is pretty messy.

With that being said, it should work fairly well to provide a command line interface for Phillips Hue brudges.

### Features and Examples
  - **interactive readline shell**
    - run `go run ./ shell` || `ziggs shell`
  - **manage multiple hue bridges at the same time**
    - e.g target specific bridge: `use ECC0FAFFFED55555`
  - **control color/saturation/hue/brightness/power per light or per group**
    - e.g. group: `set group kayos brightness 55`
    - e.g. light: `set light kayos_lamp off`
  - **list**, **delete**, and **rename** for the following targets
    - lights, groups, scenes, rules, schedules
  - **create groups**
    - e.g: `create group bedroom 5 3 2 10`
  - **specify color by HTML hex colors**
    - e.g: `set group kayos color #2eebd3`
  - **set light/group colors dynamically based on CPU load (run second time to turn off)**
    - mode 1 - average across all cores: `set group kayos cpu`
    - mode 2 (group only) cycle through individual lights in group and set based on per-core usage: `set group kayos cpu2`
  - **access firewalled bridge via SOCKS proxy**
    - to use this, change the config manually (~/.config/ziggs/config.toml)
  - **port scan to find offline (no call home) bridges on LAN**
    - see gif above for demonstration
    - config will automatically save when a bridge connection is established
  - **trigger firmware updates for bridge and lights manually**
    - e.g: `upgrade`
---

