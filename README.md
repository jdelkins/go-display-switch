# go-display-switch

Would you like to be able to push a button and switch your keyboard, mouse, video,
sound, etc. to a secondary computer, as you might with an expensive KVM device,
but without the cash outlay and spaghetti wiring? Do you have, or can afford,
a $20 usb switch like [this
one](https://www.amazon.com/UGREEN-Selector-Computers-Peripheral-One-Button/dp/B01MXXQKGM/ref=sxin_14_ac_d_bv?ac_md=0-0-QnVkZ2V0IFBpY2s%3D-ac_d_bv_bv_bv&content-id=amzn1.sym.14453ffd-7768-40d0-9a7f-8d0063113f56%3Aamzn1.sym.14453ffd-7768-40d0-9a7f-8d0063113f56&cv_ct_cx=usb+switch&keywords=usb+switch&pd_rd_i=B01MXXQKGM&pd_rd_r=397a74ce-7a29-4535-81e8-0229372aec97&pd_rd_w=ITwOt&pd_rd_wg=titwe&pf_rd_p=14453ffd-7768-40d0-9a7f-8d0063113f56&pf_rd_r=TT8GY7PQQ84RACXQVK2K&psc=1&qid=1655939775&sr=1-1-270ce31b-afa8-499f-878b-3bb461a9a5a6)?
One way to solve this problem on Linux is to use [udev][] to look for your
keyboard and mouse, and then use a command linke [ddcutil][] to switch your
monitor's input. These USB switches work by electrically
connecting/disconnecting a keyboard and mouse from one output to another one,
so the physical button, which results in a USB device appearing/disappearing,
can be interpreted as a signal to switch your monitor's inputs. This is not my
idea, but I stole it fair and square.

This project is a cut-down reimplementation of [this rust
utility](https://github.com/haimgel/display-switch), with only a small fraction
of the features, but works well enough for me. I had problems getting that
project to work for me on Linux, and, simple as the problem is, I decided to
reimplement it in [Go][]. The most important missing feature is the necessary
DDC/CI stuff; instead I favor relying on a [stand-alone utility][ddcutil] for
that. What's left is the udev interface, and luckily [that nut has been
cracked](https://github.com/pilebones/go-udev.git) as well.

Run any command on attach or detach of identified USB devices reported by
[udev][]. You can specify a command to run on attach and one on detach. Either
command is optional, but if you don't specify either one, this utility is
pointless. In my use case, I use [ddcutil][] to control my monitor's chosen
input when my keyboard is switched away using a cheap usb switch. See below for
configuration details and examples.

Why not just use udev rules? That's certainly possible, and there's nothing
wrong with that approach. I happen to find udev rule syntax a little arcane,
and I prefer not to clutter my system configuration for something like this,
which is, for me, a user-specific configuration goal.

## Installation

```
go install github.com/jdelkins/go-display-switch
```

## Basic Usage

Specify one or more devices to look for using `--rules`.

Run any command using `--add-command` and `--remove-command` options. The
commands will be run with `/bin/sh -c` and, therefore, you are free to use
constructs like environment variables and shell wildcards.

Options can be spelled out on the command line, or put into a config file. Read
on for more details.

### Examples

```
go-display-switch --config ./display-switch.toml
```

```
go-display-switch --rules='
    [{
        "action": "add|remove",
        "env": {"ID_VENDOR_ID": "445a", "ID_MODEL_ID": "2260"}
    }]' \
    --add-command "ddcutil -g DEL setvcp 60 0x0f" \
    --remove-command "ddcutil -g DEL setvcp 60 0x11"
```

Both of these commands (with the example config file provided) will look for an
input device with vendor:model "445a:2660" and switch a connected Dell monitor
input to DisplayPort-1 (`0x0f`) on connect, and to HDMI-1 (`0x11`) on
disconnect. You can use `ddcutil probe` to get clues about which VCP values are
controllable via DDC/CI, and what the accepted values are.

### Debounce

As many peripherals, especially modern keyboards and mice, present multiple
input devices, there is a good chance that attaching a device would trigger
a series of udev events that would be caught by a given set of filter rules.
When this happens, you very probably don't want to run the same command
multiple times! Accordingly, I introduce a configurable debounce time
(`--debounce-window`), intended to limit the executed command to a single
physical add or remove event. This means that you will need to wait at least
this long between detaching and attaching a device in order to execute both
commands. The default value of 500ms should be safe, and could probably be much
shorter for most devices.

### Configuration/Options

All options can be configured either from the command line or from
a configuration file. Command line options override anyting in the config file.
The configuration file can be in any of the various formats supported by the
[viper][] library. The file must have the base name `display-switch`, and is
looked for in `$XDG_CONFIG_HOME/display-switch/` (normally
`$HOME/.config/display-switch/`) or in `/etc/display-switch/`. The filename
suffix determines the parsed format, e.g., `json`, `yaml`, `toml`, `ini`, etc.
Refer to the included [example](./display-switch.toml) for the basic
configuration file format. In particular, you'll need to define a set of
`rules` to identify a device to watch for, and commands to execute when this
device is added or removed. Strings in these rules are interpreted as [regular
expressions](https://gobyexample.com/regular-expressions), so you can easily
match multiple devices or event actions in one rule. You may also provide
multiple rules[^1], if that's easier.

If `--rules` is provided on the command line, it is parsed as `json`,
regardless of the configuration file format. This json string must be a list of
dicts, each having an `action` and `env` fields. Each `env` field is itself
a dict that matches information provided in the event, as udev rules do, except
using [regular expressions].

[^1]: The `rules` configuration is defined as an array. Often one rule is
  sufficient, but you may provide more than one. Any matching rules are
  processed at runtime (boolean *or* logic).

| Flag                           | Description                                                                   |
|--------------------------------|-------------------------------------------------------------------------------|
| `-a`, `--add-command` *cmd*    | Command to run when matching device is connected/added                        |
| `-c`, `--config` *path*        | Pathname of configuration file                                                |
| `--debounce-window` *duration* | How long to wait after an event before processing more events (default 500ms) |
| `-d`, `--debug`                | Print extra debugging information                                             |
| `-r`, `--remove-command` *cmd* | Command to run when matching device is disconnected/removed                   |
| `--rules` *json*               | JSON string defining device event matching rules                              |

## Author

Joel Elkins [@jdelkins](https://github.com/jdelkins)

## License

Copyright (c) 2022 Joel D. Elkins. All rights reserved except as hereafter
provided.

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

[Go]: https://go.dev/
[ddcutil]: https://www.ddcutil.com/
[viper]: https://pkg.go.dev/github.com/spf13/viper
[udev]: https://wiki.archlinux.org/title/udev
