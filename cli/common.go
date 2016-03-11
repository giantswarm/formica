package cli

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/giantswarm/inago/controller"
	"github.com/giantswarm/inago/fleet"
	"github.com/giantswarm/inago/task"
)

var (
	atExp    = regexp.MustCompile("@")
	groupExp = regexp.MustCompile("@(.*)")
)

var (
	statusHeader = "Group | Units | FDState | FCState | SAState {{if .Verbose}}| Hash {{end}}| IP | Machine"
	statusBody   = "{{.Group}}{{if .UnitState.SliceID}}@{{.UnitState.SliceID}}{{end}} | {{.UnitState.Name}} | {{.UnitState.Desired}} | {{.UnitState.Current}} | " +
		"{{.MachineState.SystemdActive}}{{if .Verbose}} | {{.MachineState.UnitHash}}{{end}} | {{if .MachineState.IP}}{{.MachineState.IP}}{{else}}-{{end}} | {{.MachineState.ID}}"
)

func createStatus(group string, usl controller.UnitStatusList) ([]string, error) {
	if !globalFlags.Verbose {
		var err error
		usl, err = usl.Group()
		if err != nil {
			return nil, maskAny(err)
		}
	}

	out := bytes.NewBufferString("")

	header := template.Must(template.New("header").Parse(statusHeader))
	header.Execute(out, struct {
		Verbose bool
	}{
		globalFlags.Verbose,
	})
	out.WriteString("\n\n")
	tmpl := template.Must(template.New("row-format").Parse(statusBody))

	addRow := func(group string, us, ms interface{}) {
		tmpl.Execute(out, struct {
			Verbose      bool
			Group        string
			UnitState    interface{}
			MachineState interface{}
		}{
			globalFlags.Verbose,
			group,
			us,
			ms,
		})
		out.WriteString("\n")
	}

	for _, us := range usl {
		if len(us.Machine) == 0 {
			addRow(group, us,
				fleet.MachineStatus{
					ID:            "-",
					IP:            net.IP{},
					SystemdActive: "-",
					SystemdSub:    "-",
					UnitHash:      "-",
				})
		}
		for _, ms := range us.Machine {
			addRow(group, us, ms)
		}
	}

	return strings.Split(out.String(), "\n"), nil
}

type blockWithFeedbackCtx struct {
	Request    controller.Request
	Descriptor string
	NoBlock    bool
	TaskID     string
	Closer     chan struct{}
}

func maybeBlockWithFeedback(ctx blockWithFeedbackCtx) {
	if !ctx.NoBlock {
		taskObject, err := newController.WaitForTask(ctx.TaskID, ctx.Closer)
		if err != nil {
			fmt.Printf("%#v\n", maskAny(err))
			os.Exit(1)
		}

		if task.HasFailedStatus(taskObject) {
			if ctx.Request.SliceIDs == nil {
				fmt.Printf("Failed to %s group '%s'. (%s)\n", ctx.Descriptor, ctx.Request.Group, taskObject.Error)
			} else if len(ctx.Request.SliceIDs) == 0 {
				fmt.Printf("Failed to %s all slices of group '%s'. (%s)\n", ctx.Descriptor, ctx.Request.Group, taskObject.Error)
			} else {
				fmt.Printf("Failed to %s %d slices for group '%s': %v. (%s)\n", ctx.Descriptor, len(ctx.Request.SliceIDs), ctx.Request.Group, ctx.Request.SliceIDs, taskObject.Error)
			}
			os.Exit(1)
		}
	}

	if ctx.Request.SliceIDs == nil {
		fmt.Printf("Succeeded to %s group '%s'.\n", ctx.Descriptor, ctx.Request.Group)
	} else if len(ctx.Request.SliceIDs) == 0 {
		fmt.Printf("Succeeded to %s all slices of group '%s'.\n", ctx.Descriptor, ctx.Request.Group)
	} else {
		fmt.Printf("Succeeded to %s %d slices for group '%s': %v.\n", ctx.Descriptor, len(ctx.Request.SliceIDs), ctx.Request.Group, ctx.Request.SliceIDs)
	}
}
