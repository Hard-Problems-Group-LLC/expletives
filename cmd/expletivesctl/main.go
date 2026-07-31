// expletivesctl is the reference command-line controller for the local
// expletives automation protocol.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hard-Problems-Group-LLC/expletives/automation"
	"github.com/Hard-Problems-Group-LLC/expletives/internal/buildinfo"
)

const (
	exitOK            = 0
	exitUsage         = 2
	exitRejected      = 3
	exitIndeterminate = 4
	exitFailure       = 1

	defaultDialTimeout      = 5 * time.Second
	defaultOperationTimeout = 35 * time.Second
)

type options struct {
	socketPath  string
	requestID   string
	dialTimeout time.Duration
	timeout     time.Duration
	pretty      bool
	version     bool
	help        bool
}

type invocation struct {
	name string
	args []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	configuration, command, status := parseOptions(args, stderr)
	if status != exitOK {
		return status
	}
	if configuration.version {
		fmt.Fprintf(stdout, "expletivesctl %s\n", buildinfo.Current().String())
		return exitOK
	}
	if configuration.help {
		return exitOK
	}
	if command.name == "" {
		printUsage(stderr)
		return exitUsage
	}
	if configuration.socketPath == "" {
		fmt.Fprintln(stderr, "expletivesctl: --socket is required")
		return exitUsage
	}
	if configuration.timeout <= 0 {
		fmt.Fprintln(stderr, "expletivesctl: --timeout must be positive")
		return exitUsage
	}
	if configuration.dialTimeout <= 0 {
		fmt.Fprintln(stderr, "expletivesctl: --dial-timeout must be positive")
		return exitUsage
	}

	dialContext, cancelDial := context.WithTimeout(context.Background(), configuration.dialTimeout)
	client, err := automation.Dial(dialContext, configuration.socketPath)
	cancelDial()
	if err != nil {
		return reportError(stderr, configuration.socketPath, err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "expletivesctl: close: %v\n", closeErr)
		}
	}()

	if command.name == "hello" {
		if len(command.args) != 0 {
			fmt.Fprintln(stderr, "expletivesctl: hello takes no arguments")
			return exitUsage
		}
		if err := writeJSON(stdout, client.Hello(), configuration.pretty); err != nil {
			return reportError(stderr, configuration.socketPath, err)
		}
		return exitOK
	}

	ctx, cancel := context.WithTimeout(context.Background(), configuration.timeout)
	defer cancel()
	completions, err := execute(ctx, client, configuration, command, stderr)
	if err != nil {
		return reportError(stderr, configuration.socketPath, err)
	}
	var output any = completions[0]
	if len(completions) > 1 {
		output = completions
	}
	if err := writeJSON(stdout, output, configuration.pretty); err != nil {
		return reportError(stderr, configuration.socketPath, err)
	}
	for _, completion := range completions {
		switch completion.Outcome {
		case automation.OutcomeApplied, automation.OutcomeNoOp, automation.OutcomeExited:
		default:
			return exitRejected
		}
	}
	return exitOK
}

func parseOptions(args []string, stderr io.Writer) (options, invocation, int) {
	var result options
	flags := flag.NewFlagSet("expletivesctl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(stderr) }
	flags.StringVar(
		&result.socketPath,
		"socket",
		"",
		"absolute Unix-socket path advertised by expletives-test",
	)
	flags.StringVar(
		&result.requestID,
		"request-id",
		"",
		"caller-selected request ID (never automatically retried)",
	)
	flags.DurationVar(
		&result.dialTimeout,
		"dial-timeout",
		defaultDialTimeout,
		"connection and hello deadline",
	)
	flags.DurationVar(
		&result.timeout,
		"timeout",
		defaultOperationTimeout,
		"operation deadline",
	)
	flags.BoolVar(&result.pretty, "pretty", true, "indent JSON output")
	flags.BoolVar(&result.version, "version", false, "print build provenance")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			result.help = true
			return result, invocation{}, exitOK
		}
		return options{}, invocation{}, exitUsage
	}
	remaining := flags.Args()
	if result.version {
		if len(remaining) != 0 {
			fmt.Fprintln(stderr, "expletivesctl: --version takes no command")
			return options{}, invocation{}, exitUsage
		}
		return result, invocation{}, exitOK
	}
	if len(remaining) == 0 {
		return result, invocation{}, exitOK
	}
	return result, invocation{name: remaining[0], args: remaining[1:]}, exitOK
}

func execute(
	ctx context.Context,
	client *automation.Client,
	configuration options,
	command invocation,
	stderr io.Writer,
) ([]automation.Completion, error) {
	switch command.name {
	case "snapshot", "observe":
		sequence, err := parseSnapshotArgs(command.args, stderr)
		if err != nil {
			return nil, err
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.Observe(ctx, requestID, sequence)
		return one(completion, err)
	case "wait":
		after, err := parseWaitArgs(command.args, stderr)
		if err != nil {
			return nil, err
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.WaitSnapshot(ctx, requestID, after)
		return one(completion, err)
	case "key":
		event, err := parseKeyArgs(command.args)
		if err != nil {
			return nil, err
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.InjectInput(ctx, requestID, event)
		return one(completion, err)
	case "keys":
		events, err := parseKeyTokens(command.args)
		if err != nil {
			return nil, err
		}
		completions := make([]automation.Completion, 0, len(events))
		for index, event := range events {
			requestID, err := indexedRequestID(configuration.requestID, index, len(events))
			if err != nil {
				return nil, err
			}
			completion, err := client.InjectInput(ctx, requestID, event)
			if err != nil {
				return nil, err
			}
			completions = append(completions, completion)
		}
		return completions, nil
	case "command":
		if len(command.args) < 1 || len(command.args) > 2 {
			return nil, errors.New("command usage: command COMMAND [TARGET_KEY]")
		}
		targetKey := ""
		if len(command.args) == 2 {
			targetKey = command.args[1]
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.InvokeCommand(
			ctx,
			requestID,
			command.args[0],
			targetKey,
		)
		return one(completion, err)
	case "result":
		if len(command.args) != 1 {
			return nil, errors.New("result usage: result TARGET_REQUEST_ID")
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.QueryResult(
			ctx,
			requestID,
			command.args[0],
		)
		return one(completion, err)
	case "reset-input":
		if len(command.args) != 0 {
			return nil, errors.New("reset-input takes no arguments")
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.ResetInput(ctx, requestID)
		return one(completion, err)
	case "shutdown":
		if len(command.args) != 0 {
			return nil, errors.New("shutdown takes no arguments")
		}
		requestID, err := requestID(configuration.requestID)
		if err != nil {
			return nil, err
		}
		completion, err := client.Shutdown(ctx, requestID)
		return one(completion, err)
	default:
		return nil, fmt.Errorf("unknown command %q", command.name)
	}
}

func parseSnapshotArgs(args []string, stderr io.Writer) (*uint64, error) {
	flags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var sequence uint64
	flags.Uint64Var(&sequence, "sequence", 0, "exact retained frame sequence")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if flags.NArg() != 0 {
		return nil, errors.New("snapshot usage: snapshot [--sequence N]")
	}
	if sequence == 0 {
		return nil, nil
	}
	return &sequence, nil
}

func parseWaitArgs(args []string, stderr io.Writer) (uint64, error) {
	flags := flag.NewFlagSet("wait", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var after uint64
	flags.Uint64Var(&after, "after", 0, "wait for a frame newer than this sequence")
	if err := flags.Parse(args); err != nil {
		return 0, err
	}
	if flags.NArg() != 0 {
		return 0, errors.New("wait usage: wait [--after N]")
	}
	return after, nil
}

func parseKeyArgs(args []string) (automation.KeyEvent, error) {
	if len(args) != 2 {
		return automation.KeyEvent{}, errors.New("key usage: key down|up|press KEY")
	}
	kind, err := keyKind(args[0])
	if err != nil {
		return automation.KeyEvent{}, err
	}
	if args[1] == "" {
		return automation.KeyEvent{}, errors.New("key identity must not be empty")
	}
	return automation.KeyEvent{Kind: kind, Key: args[1]}, nil
}

func parseKeyTokens(args []string) ([]automation.KeyEvent, error) {
	if len(args) == 0 {
		return nil, errors.New(
			"keys usage: keys down:KEY|up:KEY|press:KEY [...]",
		)
	}
	result := make([]automation.KeyEvent, 0, len(args))
	for _, token := range args {
		kindText, key, found := strings.Cut(token, ":")
		if !found || key == "" {
			return nil, fmt.Errorf("invalid key token %q; expected KIND:KEY", token)
		}
		kind, err := keyKind(kindText)
		if err != nil {
			return nil, err
		}
		result = append(result, automation.KeyEvent{Kind: kind, Key: key})
	}
	return result, nil
}

func keyKind(value string) (string, error) {
	switch value {
	case "down", automation.KeyDown:
		return automation.KeyDown, nil
	case "up", automation.KeyUp:
		return automation.KeyUp, nil
	case "press", automation.KeyPress:
		return automation.KeyPress, nil
	default:
		return "", fmt.Errorf(
			"invalid key event kind %q; use down, up, or press",
			value,
		)
	}
}

func requestID(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	return automation.NewRequestID()
}

func indexedRequestID(configured string, index, count int) (string, error) {
	if configured == "" {
		return automation.NewRequestID()
	}
	if count == 1 {
		return configured, nil
	}
	return configured + "-" + strconv.Itoa(index+1), nil
}

func one(completion automation.Completion, err error) ([]automation.Completion, error) {
	if err != nil {
		return nil, err
	}
	return []automation.Completion{completion}, nil
}

func writeJSON(writer io.Writer, value any, pretty bool) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	return nil
}

func reportError(stderr io.Writer, socketPath string, err error) int {
	fmt.Fprintf(stderr, "expletivesctl: %v\n", err)
	var indeterminate *automation.IndeterminateError
	if errors.As(err, &indeterminate) {
		fmt.Fprintf(
			stderr,
			"expletivesctl: request ID %s; query_result recovery: expletivesctl --socket %s result %s\n",
			shellQuote(indeterminate.RequestID),
			shellQuote(socketPath),
			shellQuote(indeterminate.RequestID),
		)
		return exitIndeterminate
	}
	return exitFailure
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, `Usage:
  expletivesctl [GLOBAL OPTIONS] hello
  expletivesctl [GLOBAL OPTIONS] snapshot [--sequence N]
  expletivesctl [GLOBAL OPTIONS] wait [--after N]
  expletivesctl [GLOBAL OPTIONS] key down|up|press KEY
  expletivesctl [GLOBAL OPTIONS] keys down:KEY|up:KEY|press:KEY [...]
  expletivesctl [GLOBAL OPTIONS] command COMMAND [TARGET_KEY]
  expletivesctl [GLOBAL OPTIONS] result TARGET_REQUEST_ID
  expletivesctl [GLOBAL OPTIONS] reset-input
  expletivesctl [GLOBAL OPTIONS] shutdown

Global options must precede the command:
  --socket PATH       explicit Unix-socket path
  --request-id ID     caller-selected request ID
  --dial-timeout DURATION
                      connection and hello deadline (default 5s)
  --timeout DURATION  operation deadline (default 35s)
  --pretty=BOOL       indent JSON output (default true)
  --version           print build provenance

Requests are never automatically retried. A disconnect after submission may
leave the outcome indeterminate; query a retained result from a new connection
only when you know the original request ID.`)
}
