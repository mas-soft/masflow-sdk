// Workflow Client example -- execute, monitor, and manage workflows from Go.
//
// This sample demonstrates the WorkflowClient as a standalone CLI tool
// (no Temporal worker needed). It can execute workflows, check status,
// list running workflows, and perform lifecycle operations.
//
//	go run . --url=http://localhost:10000 execute --yaml workflow.yaml
//	go run . --url=http://localhost:10000 status <workflow-id>
//	go run . --url=http://localhost:10000 monitor <workflow-id>
//	go run . --url=http://localhost:10000 list
//	go run . --url=http://localhost:10000 cancel <workflow-id>
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/mas-soft/masflow/sdk"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	// Parse --url flag manually for simplicity
	url := "http://localhost:10000"
	args := os.Args[1:]
	if args[0] == "--url" || args[0] == "-url" {
		if len(args) < 3 {
			printUsage()
			os.Exit(1)
		}
		url = args[1]
		args = args[2:]
	}

	wc := sdk.NewWorkflowClient(url)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "execute":
		cmdExecute(ctx, wc, cmdArgs)
	case "status":
		cmdStatus(ctx, wc, cmdArgs)
	case "describe":
		cmdDescribe(ctx, wc, cmdArgs)
	case "monitor":
		cmdMonitor(ctx, wc, cmdArgs)
	case "list":
		cmdList(ctx, wc)
	case "search":
		cmdSearch(ctx, wc, cmdArgs)
	case "cancel":
		cmdCancel(ctx, wc, cmdArgs)
	case "pause":
		cmdPause(ctx, wc, cmdArgs)
	case "resume":
		cmdResume(ctx, wc, cmdArgs)
	case "signal":
		cmdSignal(ctx, wc, cmdArgs)
	case "validate":
		cmdValidate(ctx, wc, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func cmdExecute(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: execute --yaml <file>")
	}

	var yamlFile string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--yaml" {
			yamlFile = args[i+1]
		}
	}
	if yamlFile == "" {
		log.Fatal("--yaml flag is required")
	}

	data, err := os.ReadFile(yamlFile)
	if err != nil {
		log.Fatalf("Read %s: %v", yamlFile, err)
	}

	result, err := wc.ExecuteYAML(ctx, string(data), &sdk.ExecuteSourceOptions{
		WorkflowID: fmt.Sprintf("cli-%d", time.Now().Unix()),
	})
	if err != nil {
		log.Fatalf("Execute: %v", err)
	}
	fmt.Printf("Workflow started:\n")
	fmt.Printf("  ID:     %s\n", result.WorkflowID)
	fmt.Printf("  Status: %s\n", result.Status)
}

func cmdStatus(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: status <workflow-id>")
	}
	status, err := wc.GetStatus(ctx, args[0])
	if err != nil {
		log.Fatalf("GetStatus: %v", err)
	}
	fmt.Printf("Workflow: %s\n", status.WorkflowID)
	fmt.Printf("Status:   %s\n", status.Status)
	if len(status.Trace) > 0 {
		fmt.Printf("Trace (%d entries):\n", len(status.Trace))
		for _, t := range status.Trace {
			fmt.Printf("  [%s] %s: %s - %s\n",
				t.Timestamp.Format(time.RFC3339), t.StepType, t.Status, t.Details)
			if t.Error != "" {
				fmt.Printf("    Error: %s\n", t.Error)
			}
		}
	}
}

func cmdDescribe(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: describe <workflow-id> [run-id]")
	}
	workflowID := args[0]
	runID := ""
	if len(args) > 1 {
		runID = args[1]
	}

	info, err := wc.Describe(ctx, workflowID, runID)
	if err != nil {
		log.Fatalf("Describe: %v", err)
	}
	fmt.Printf("Workflow:   %s\n", info.WorkflowID)
	fmt.Printf("Run ID:    %s\n", info.RunID)
	fmt.Printf("Type:      %s\n", info.WorkflowType)
	fmt.Printf("Status:    %s\n", info.Status)
	fmt.Printf("Started:   %s\n", info.StartTime.Format(time.RFC3339))
	if !info.CloseTime.IsZero() {
		fmt.Printf("Closed:    %s\n", info.CloseTime.Format(time.RFC3339))
	}
	fmt.Printf("Duration:  %s\n", info.ExecutionTime)
	fmt.Printf("Namespace: %s\n", info.Namespace)
	fmt.Printf("Queue:     %s\n", info.TaskQueue)
	fmt.Printf("Errors:    %v\n", info.HasErrors)
	if info.ErrorMessage != "" {
		fmt.Printf("Error:     %s\n", info.ErrorMessage)
	}
	if len(info.ChildWorkflows) > 0 {
		fmt.Printf("Children (%d):\n", len(info.ChildWorkflows))
		for _, c := range info.ChildWorkflows {
			fmt.Printf("  %s [%s] %s\n", c.WorkflowID, c.Status, c.WorkflowType)
		}
	}
}

func cmdMonitor(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: monitor <workflow-id> [run-id]")
	}
	workflowID := args[0]
	runID := ""
	if len(args) > 1 {
		runID = args[1]
	}

	mon, err := wc.Monitor(ctx, workflowID, runID)
	if err != nil {
		log.Fatalf("Monitor: %v", err)
	}

	fmt.Printf("Workflow: %s [%s]\n", mon.WorkflowID, mon.Status)
	fmt.Printf("Started:  %s (elapsed: %dms)\n", mon.StartTime.Format(time.RFC3339), mon.ElapsedMs)
	fmt.Printf("Progress: %d/%d steps (%.0f%%)\n",
		mon.Progress.CurrentStep, mon.Progress.TotalSteps, mon.Progress.Percentage)
	fmt.Printf("Counts:   total=%d pending=%d running=%d completed=%d failed=%d\n",
		mon.StepCounts.Total, mon.StepCounts.Pending, mon.StepCounts.Running,
		mon.StepCounts.Completed, mon.StepCounts.Failed)

	if len(mon.Steps) > 0 {
		fmt.Printf("\nSteps:\n")
		for _, s := range mon.Steps {
			dur := ""
			if s.DurationMs > 0 {
				dur = fmt.Sprintf(" (%dms)", s.DurationMs)
			}
			errStr := ""
			if s.Error != "" {
				errStr = fmt.Sprintf(" ERROR: %s", s.Error)
			}
			fmt.Printf("  %d. [%s] %s (%s)%s%s\n",
				s.Index, s.Status, s.Name, s.StepType, dur, errStr)
		}
	}
	if mon.LastError != "" {
		fmt.Printf("\nLast error: %s\n", mon.LastError)
	}
}

func cmdList(ctx context.Context, wc *sdk.WorkflowClient) {
	list, err := wc.List(ctx, &sdk.ListWorkflowsOptions{
		PageSize: 20,
		OrderBy:  "start_time desc",
	})
	if err != nil {
		log.Fatalf("List: %v", err)
	}

	fmt.Printf("Workflows (showing %d of %d):\n\n", len(list.Workflows), list.TotalCount)
	for _, w := range list.Workflows {
		dur := ""
		if w.ExecutionTime > 0 {
			dur = fmt.Sprintf(" (%s)", w.ExecutionTime)
		}
		fmt.Printf("  %-40s [%-10s] %s%s\n", w.WorkflowID, w.Status, w.WorkflowType, dur)
	}
}

func cmdSearch(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}
	results, err := wc.Search(ctx, &sdk.SearchWorkflowsOptions{
		Query:    query,
		PageSize: 20,
	})
	if err != nil {
		log.Fatalf("Search: %v", err)
	}

	fmt.Printf("Search results (%d of %d):\n\n", len(results.Workflows), results.TotalCount)
	for _, w := range results.Workflows {
		tags := ""
		if len(w.Tags) > 0 {
			tags = fmt.Sprintf(" [%s]", w.Tags)
		}
		fmt.Printf("  %-30s %-12s %s%s\n", w.WorkflowID, w.Status, w.Name, tags)
	}
}

func cmdCancel(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: cancel <workflow-id> [reason]")
	}
	reason := "cancelled via CLI"
	if len(args) > 1 {
		reason = args[1]
	}
	if err := wc.Cancel(ctx, args[0], reason); err != nil {
		log.Fatalf("Cancel: %v", err)
	}
	fmt.Printf("Workflow %s cancel requested.\n", args[0])
}

func cmdPause(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: pause <workflow-id> [run-id]")
	}
	runID := ""
	if len(args) > 1 {
		runID = args[1]
	}
	if err := wc.Pause(ctx, args[0], runID, "paused via CLI"); err != nil {
		log.Fatalf("Pause: %v", err)
	}
	fmt.Printf("Workflow %s paused.\n", args[0])
}

func cmdResume(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: resume <workflow-id> [run-id]")
	}
	runID := ""
	if len(args) > 1 {
		runID = args[1]
	}
	if err := wc.Resume(ctx, args[0], runID, "resumed via CLI"); err != nil {
		log.Fatalf("Resume: %v", err)
	}
	fmt.Printf("Workflow %s resumed.\n", args[0])
}

func cmdSignal(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: signal <workflow-id> <signal-name>")
	}
	result, err := wc.Signal(ctx, args[0], args[1], nil)
	if err != nil {
		log.Fatalf("Signal: %v", err)
	}
	fmt.Printf("Signal %q sent to workflow %s (run: %s)\n", args[1], result.WorkflowID, result.RunID)
}

func cmdValidate(ctx context.Context, wc *sdk.WorkflowClient, args []string) {
	if len(args) < 1 {
		log.Fatal("Usage: validate <yaml-file>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		log.Fatalf("Read %s: %v", args[0], err)
	}

	result, err := wc.Validate(ctx, string(data))
	if err != nil {
		log.Fatalf("Validate: %v", err)
	}

	if result.Valid {
		fmt.Println("Workflow is valid.")
	} else {
		fmt.Println("Workflow is INVALID:")
		for _, e := range result.Errors {
			fmt.Printf("  ERROR: %s\n", e)
		}
	}
	for _, w := range result.Warnings {
		fmt.Printf("  WARNING: %s\n", w)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: workflow-client [--url <platform-url>] <command> [args...]

Commands:
  execute  --yaml <file>           Execute a workflow from YAML
  status   <workflow-id>           Get workflow status and trace
  describe <workflow-id> [run-id]  Get detailed workflow info
  monitor  <workflow-id> [run-id]  Real-time step monitoring
  list                             List recent workflows
  search   [query]                 Search workflows
  cancel   <workflow-id> [reason]  Cancel a workflow
  pause    <workflow-id> [run-id]  Pause a workflow
  resume   <workflow-id> [run-id]  Resume a paused workflow
  signal   <workflow-id> <name>    Send a signal to a workflow
  validate <yaml-file>             Validate workflow YAML
`)
}
