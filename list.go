package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"text/tabwriter"

	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	format string
)

func addListFlags(fs *flag.FlagSet) {
	fs.StringVar(&format, "f", "{{.Importpath}}\t{{.Repository}}{{.Path}}\t{{.Branch}}\t{{.Revision}}", "format template")
}

var cmdList = &Command{
	Name:      "list",
	UsageLine: "list [-f format]",
	Short:     "list dependencies one per line",
	Long: `list formats the contents of the manifest file.

Flags:
	-f
		controls the template used for printing each manifest entry. If not supplied
		the default value is "{{.Importpath}}\t{{.Repository}}{{.Path}}\t{{.Branch}}\t{{.Revision}}"

`,
	Run: func(args []string) error {
		m, err := vendor.ReadManifest(manifestFile)
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			return fmt.Errorf("unable to parse template %q: %v", format, err)
		}
		w := tabwriter.NewWriter(os.Stdout, 1, 2, 1, ' ', 0)
		for _, dep := range m.Dependencies {
			if err := tmpl.Execute(w, dep); err != nil {
				return fmt.Errorf("unable to execute template: %v", err)
			}
			fmt.Fprintln(w)
		}
		return w.Flush()
	},
	AddFlags: addListFlags,
}
