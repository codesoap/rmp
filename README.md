# Richard's Music Player
rmp is a minimal OpenSubsonic terminal music player. Through fuzzy
searching the library it allows for frictionless usage and supports
last.fm (or similar) integration by scrobbling and queuing similar
songs.

## Demo

https://github.com/user-attachments/assets/00776d83-b093-45d3-8ee6-2f3fad405136

## Installation
The `ffmpeg` tool needs to be installed for rmp to work.

You can download precompiled binaries from the [releases
page](https://github.com/codesoap/rmp/releases) or install it with `go
install github.com/codesoap/rmp/cmd/rmp@latest`.

## Usage
```console
$ rmp -h
Usage: rmp [-p]

Within rmp, press ? to view the key map.

A configuration is expected at '/home/richard/.config/rmp/rmp.conf'.
Create a configuration file with the following content:
    server-url = <subsonic-server-url>
    user       = <your-username>

Example:
    server-url = http://192.168.1.12:4747
    user       = admin

Songs are cached locally at '/home/richard/.cache/rmp/songs'.
When no song is playing, you may delete them.

Options:
  -p    change previously entered password
```
