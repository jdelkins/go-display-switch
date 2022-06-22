# go-display-switch

A cut-down reimplementation of [this
project](https://github.com/haimgel/display-switch), with only a small fraction
of the features, but works well enough for me. Implemented in [Go][].

Run any command on attach or detach of a certain USB device, identified by
Vendor ID and Model ID. The commands to run are given as command line
parameters. In my use case, I use [ddcutil][] to control my monitor's chosen
input when my keyboard is switched away using my cheap usb kvm. See below.

[Go]: https://go.dev/
[ddcutil]: https://www.ddcutil.com/

## Installation

```
    go install github.com/jdelkins/go-display-switch
```

## Usage

Run any command using the optional `-connect` and `-disconnect` command line
flags. The command will be run with `/bin/sh -c` and, therefore, you can
constructs like environment variables and shell wildcards.

### Example
```
    go-display-switch -vendorid 445a -modelid 2260 \
        -connect "ddcutil -g DEL setvcp 60 0x0f" \
        -disconnect "ddcutil -g DEL setvcp 60 0x11"
```

This will look for an input device with vendor:model "445a:2660" and switch my
Dell monitor input to DisplayPort-1 (`0x0f`) on connect, and to HDMI-1 (`0x11`)
on disconnect. You can use `ddcutil probe` to get clues about which VCP values
are controllable via DDC/CI, and what the accepted values are.

### Options description

```
  -connect string
    	Command to execute on connect. Will be run with 'sh -c'
  -disconnect string
    	Command to execute on disconnect. Will be run with 'sh -c'
  -modelid string
    	Product ID of monitored usb device
  -vendorid string
    	Vendor ID of monitored usb device
```
## License

Copyright (c) 2022 Joel D. Elkins. All rights reserved other than as hereafter
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
