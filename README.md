# polybar-pomo

Simple Pomodoro Widget for [Polybar](https://github.com/polybar/polybar)

### Requirements

Should work in any Linux OS with the following packages:

- `go`
- `openbsd-netcat` or `socat`

### Installation

```bash
git clone https://github.com/neumann-mlucas/polybar-pomo
cd polybar-pomo
go build polybar-pomo.go
# Copy the binary to your Polybar config directory or put it in your $PATH
cp polybar-pomo $HOME/.config/polybar
```

### Polybar Configuration Example

Using `netcat`:

```
[module/polybar-pomo]
type = custom/script

exec = ~/.config/polybar/polybar-pomo
tail = true

label = %output%
label-padding = 4
click-left = echo "pause" | nc -w 1 -U /tmp/polybar-pomo.sock
click-right = echo "toggle" | nc -w 1 -U /tmp/polybar-pomo.sock
scroll-up = echo "inc" | nc -w 1 -U /tmp/polybar-pomo.sock
scroll-down = echo "dec" | nc -w 1 -U /tmp/polybar-pomo.sock
```

Using `socat`:

```
[module/polybar-pomo]
type = custom/script

exec = ~/.config/polybar/polybar-pomo
tail = true

label = %output%
label-padding = 4
click-left = echo "pause" | socat - UNIX-CONNECT:/tmp/polybar-pomo
click-right = echo "toggle" | socat - UNIX-CONNECT:/tmp/polybar-pomo
scroll-up = echo "inc" | socat - UNIX-CONNECT:/tmp/polybar-pomo
scroll-down = echo "dec" | socat - UNIX-CONNECT:/tmp/polybar-pomo
```

#### Change Default Work and Rest Times

Pass `-w` (work time) and `-r` (rest time) in the exec line in your Polybar config.

```
exec = ~/.config/polybar/polybar-pomo -w 5 -p 25
```
