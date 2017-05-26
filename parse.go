package cli

import (
	"fmt"
	"github.com/silvertern/cli/command"
	"github.com/silvertern/cli/option"
	"path"
	"strconv"
	"strings"
)

const (
	helpKey  = "help"
	helpChar = 'h'
	trueStr  = "true"
)

func Parse(a App, appargs []string) (invocation []string, args []string, opts map[string]string, err error) {
	_, appname := path.Split(appargs[0])

	invocation, argsAndOpts, expArgs, accptOpts := evalCommand(a, appargs[1:])
	invocation = append([]string{appname}, invocation...)

	if args, opts, err = splitArgsAndOpts(argsAndOpts, accptOpts); err == nil {
		if _, ok := opts["help"]; !ok {
			if err = assertArgs(expArgs, args); err == nil {
				err = assertOpts(accptOpts, opts)
			}
		}
	}
	return invocation, args, opts, err
}

func evalCommand(a App, appargs []string) (invocation []string, argsAndOpts []string, expArgs []command.Arg, accptOpts []option.Option) {
	invocation = []string{}
	argsAndOpts = appargs
	expArgs = a.Args()
	accptOpts = a.Options()

	cmds2check := a.Commands()
	for i, arg := range appargs {
		matched := false
		for _, cmd := range cmds2check {
			if cmd.Key() == arg || cmd.Shortcut() == arg {
				invocation = append(invocation, cmd.Key())
				argsAndOpts = appargs[i+1:]
				expArgs = cmd.Args()
				accptOpts = append(accptOpts, cmd.Options()...)

				cmds2check = cmd.Commands()
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}
	return invocation, argsAndOpts, expArgs, accptOpts
}

func splitArgsAndOpts(appargs []string, accptOpts []option.Option) (args []string, opts map[string]string, err error) {
	opts = make(map[string]string)

	danglingOpt := ""
	for _, arg := range appargs {
		if danglingOpt != "" {
			opts[danglingOpt] = arg
			danglingOpt = ""
			continue
		}

		if strings.HasPrefix(arg, "--") {
			arg = arg[2:]
			if arg == helpKey {
				return nil, map[string]string{helpKey: trueStr}, nil
			}
			parts := strings.Split(arg, "=")
			key := parts[0]
			matched := false
			for _, accptOpt := range accptOpts {
				if accptOpt.Key() == key {
					if accptOpt.Type() == option.TypeBool {
						if len(parts) == 1 {
							opts[accptOpt.Key()] = trueStr
						} else {
							return args, opts, fmt.Errorf("boolean options have true assigned implicitly, found value for --%s", key)
						}
					} else if len(parts) >= 2 {
						opts[accptOpt.Key()] = strings.Join(parts[1:], "=") // permit = in values
					} else {
						return args, opts, fmt.Errorf("missing value for option --%s", key)
					}
					matched = true
					break
				}
			}
			if !matched {
				return args, opts, fmt.Errorf("unknown option --%s", key)
			}
			continue
		}

		if strings.HasPrefix(arg, "-") {
			arg = arg[1:]

			for i, char := range arg {
				if char == helpChar {
					return nil, map[string]string{helpKey: trueStr}, nil
				}
				matched := false
				for _, accptOpt := range accptOpts {
					if accptOpt.CharKey() == char {
						if accptOpt.Type() == option.TypeBool {
							opts[accptOpt.Key()] = trueStr
						} else if i == len(arg)-1 {
							danglingOpt = accptOpt.Key()
						} else {
							return args, opts, fmt.Errorf("non-boolean flag -%v in non-terminal position", string(char))
						}
						matched = true
						break
					}
				}
				if !matched {
					return args, opts, fmt.Errorf("unknown flag -%v", string(char))
				}
			}
			continue
		}

		args = append(args, arg)
	}
	if danglingOpt != "" {
		return args, opts, fmt.Errorf("dangling option --%s", danglingOpt)
	}
	return args, opts, nil
}

func assertArgs(expected []command.Arg, actual []string) error {
	if len(expected) == 0 || !expected[len(expected)-1].Optional {
		if len(expected) > len(actual) {
			return fmt.Errorf("missing required argument %v", expected[len(actual)].Key)
		} else if len(expected) < len(actual) {
			return fmt.Errorf("unknown arguments %v", actual[len(expected):])
		}
	}
	for i, e := range expected {
		if len(actual) < i+1 {
			if !e.Optional {
				return fmt.Errorf("missing required argument %s", e.Key)
			}
			break
		}
		arg := actual[i]
		switch e.Type {
		case option.TypeBool:
			if _, err := strconv.ParseBool(arg); err != nil {
				return fmt.Errorf("argument %s must be a boolean value, found %v", e.Key, arg)
			}
		case option.TypeInt:
			if _, err := strconv.ParseInt(arg, 10, 64); err != nil {
				return fmt.Errorf("argument %s must be an integer value, found %v", e.Key, arg)
			}
		case option.TypeNumber:
			if _, err := strconv.ParseFloat(arg, 64); err != nil {
				return fmt.Errorf("argument %s must be a number, found %v", e.Key, arg)
			}
		}
	}
	return nil
}

func assertOpts(permitted []option.Option, actual map[string]string) error {
	for key, value := range actual {
		for _, p := range permitted {
			if p.Key() == key {
				switch p.Type() {
				case option.TypeInt:
					if _, err := strconv.ParseInt(value, 10, 64); err != nil {
						return fmt.Errorf("option --%s must be given an integer value, found %v", p.Key(), value)
					}
				case option.TypeNumber:
					if _, err := strconv.ParseFloat(value, 64); err != nil {
						return fmt.Errorf("option --%s must must be given a number, found %v", p.Key(), value)
					}
				}
				break
			}
		}
	}
	return nil
}
