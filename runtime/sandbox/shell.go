package sandbox

import (
	"path"
	"strings"
)

var shellNames = map[string]struct{}{
	"sh": {}, "bash": {}, "zsh": {}, "dash": {}, "ash": {},
}

// IsShellCommand reports whether command invokes a shell interpreter,
// including common wrappers such as env and busybox.
func IsShellCommand(command []string) bool {
	if len(command) == 0 {
		return false
	}
	return isShellInvocation(command)
}

func isShellInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	base := path.Base(strings.TrimSpace(args[0]))
	if isShellName(base) {
		return true
	}
	switch base {
	case "env":
		return isShellInvocation(skipEnvPrefix(args[1:]))
	case "busybox":
		if len(args) > 1 {
			return isShellName(path.Base(strings.TrimSpace(args[1])))
		}
	}
	return false
}

func isShellName(name string) bool {
	_, ok := shellNames[name]
	return ok
}

func skipEnvPrefix(args []string) []string {
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			return args[i+1:]
		}
		if strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-") {
			i++
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			break
		}
		switch arg {
		case "-i", "-0", "-v", "-V", "--version", "--help":
			i++
		case "-u", "-C", "-S", "--unset":
			if i+1 >= len(args) {
				return nil
			}
			i += 2
		default:
			if strings.Contains(arg, "=") {
				i++
			} else {
				i++
			}
		}
	}
	return args[i:]
}
