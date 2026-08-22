# Richard's Music Player
rmp is a minimal OpenSubsonic terminal music player. It can rapidly
create custom queues by fuzzy searching the whole library and providing
suggestions (through services like last.fm).

It works best on Linux, but has also been tested on OpenBSD and will
likely work with other Unix-like operating systems.

## Demo

https://github.com/user-attachments/assets/f368219f-7b67-4594-8bda-12a80ead5355

## Installation
The `ffmpeg` tool needs to be installed for rmp to work. When not using Linux,
you might also have to install pulseaudio.

You can download precompiled binaries from the [releases
page](https://github.com/codesoap/rmp/releases) or install it from source:

```
go install github.com/codesoap/rmp/cmd/rmp@latest
```

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
