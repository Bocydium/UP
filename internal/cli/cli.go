package cli

// Flags holds parsed command-line flags.
type Flags struct {
	NoConfirm bool
	NoCheck   bool
	Needed    bool
	Quiet     bool
	Plan      bool
}

// ParseFlags extracts flags from args and returns the remaining positional args.
func ParseFlags(args []string) Flags {
	f := Flags{}
	for _, a := range args {
		switch a {
		case "--noconfirm":
			f.NoConfirm = true
		case "--nocheck":
			f.NoCheck = true
		case "--needed":
			f.Needed = true
		case "-q", "--quiet":
			f.Quiet = true
		case "--plan":
			f.Plan = true
		}
	}
	return f
}
