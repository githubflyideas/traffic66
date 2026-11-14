package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Every command's help has the same parts, in the same order: how to call
// it, what it does, examples, and its flags.
type cmdHelp struct {
	usage, about string
	examples     []string
}

var helps = map[string]cmdHelp{
	"serve": {"traffic66 [serve] [flags]", "Collect sFlow, NetFlow and IPFIX from your devices and serve the web UI.", []string{
		"traffic66                                         # web UI on :8066, collectors on UDP 6343, 2055, 4739",
		"traffic66 -data /var/lib/traffic66 -retention-days 90",
		"traffic66 -listen sflow=:6343,netflow=:9995       # other UDP ports",
		"traffic66 -tls                                    # HTTPS with a certificate made here",
		"traffic66 -addr 127.0.0.1:8066 -public-url https://traffic66.example.org   # behind a reverse proxy",
	}},
	"demo": {"traffic66 demo [flags]", "Run with a built-in simulated network, to try traffic66 without devices.", []string{
		"traffic66 demo",
		"traffic66 demo -addr :9066 -data /tmp/t66-demo",
	}},
	"tui": {"traffic66 tui [flags]", "Open the terminal UI of a running traffic66.", []string{
		"traffic66 tui                                     # the traffic66 on this machine",
		"traffic66 tui -server https://t66.example.org -user alice",
		"traffic66 tui -lang ja",
	}},
	"simulate": {"traffic66 simulate [flags]", "Send simulated sFlow, NetFlow and IPFIX exports to a collector, for testing.", []string{
		"traffic66 simulate -to 192.0.2.10",
	}},
	"passwd": {"traffic66 passwd [flags]", "Set, add, list or delete login users. Passwords are stored hashed in\n<data directory>/password; a running traffic66 uses changes at the next sign-in.\nAdd -data <directory> when traffic66 runs with -data.", []string{
		"traffic66 passwd                                  # set the password of admin",
		"traffic66 passwd -user alice                      # add alice, or change her password",
		"traffic66 passwd -user alice -delete              # delete alice",
		"traffic66 passwd -list                            # list the users",
		"traffic66 passwd -generate                        # set a random password and print it",
		"printf 'pw\\npw\\n' | traffic66 passwd -user bob   # from a script",
	}},
	"interfaces": {"traffic66 interfaces", "List the network interfaces usable for local capture (-capture).", []string{
		"traffic66 interfaces",
	}},
	"version": {"traffic66 version", "Print the version and the kind of build.", nil},
	"pcap": {"traffic66 FILE.pcap [FILE2.pcapng FILE3.pcap] [flags]", "Analyse capture files in the web UI, on this computer only.", []string{
		"traffic66 capture.pcapng",
		"traffic66 a.pcap b.pcap -dns",
	}},
}

// printHelp writes the help of a command; fs gives its flags, if any.
func printHelp(w io.Writer, name string, fs *flag.FlagSet) {
	h := helps[name]
	fmt.Fprintf(w, "Usage: %s\n\n%s\n", h.usage, h.about)
	if len(h.examples) > 0 {
		fmt.Fprint(w, "\nExamples:\n")
		for _, e := range h.examples {
			fmt.Fprintf(w, "  %s\n", e)
		}
	}
	n := 0
	if fs != nil {
		fs.VisitAll(func(*flag.Flag) { n++ })
	}
	if n > 0 {
		fmt.Fprint(w, "\nFlags:\n")
		out := fs.Output()
		fs.SetOutput(w)
		fs.PrintDefaults()
		fs.SetOutput(out)
	}
}

// withHelp makes fs print the command's help for -h.
func withHelp(fs *flag.FlagSet, name string) *flag.FlagSet {
	fs.Usage = func() { printHelp(os.Stderr, name, fs) }
	return fs
}

// wantsHelp reports whether args ask for help (commands without flags).
func wantsHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "-help", "--help":
			return true
		}
	}
	return false
}
