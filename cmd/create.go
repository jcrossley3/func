package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/ory/viper"
	"github.com/spf13/cobra"

	fn "github.com/boson-project/func"
	"github.com/boson-project/func/buildpacks"
	"github.com/boson-project/func/utils"
)

func init() {
	// Add to the root a new "Create" command which obtains an appropriate
	// instance of fn.Client from the given client creator function.
	root.AddCommand(NewCreateCmd(newCreateClient))
}

// newCreateClient returns an instance of fn.Client for the "Create" command.
// The createClientFn is a client factory which creates a new Client for use by
// the create command during normal execution (see tests for alternative client
// factories which return clients with various mocks).
func newCreateClient(repositories string, verbose bool) *fn.Client {
	return fn.New(fn.WithRepositories(repositories), fn.WithVerbose(verbose))
}

// createClientFn is a factory function which returns a Client suitable for
// use with the Create command.
type createClientFn func(repositories string, verbose bool) *fn.Client

// NewCreateCmd creates a create command using the given client creator.
func NewCreateCmd(clientFn createClientFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [PATH]",
		Short: "Create a function project",
		Long: `Create a function project

Creates a new function project in PATH, or in the current directory if no PATH is given. 
The name of the project is determined by the directory name the project is created in.
`,
		Example: `
# Create a Node.js function project in the current directory, choosing the
# directory name as the project's name.
kn func create

# Create a Quarkus function project in the directory "sample-service". 
# The directory will be created in the local directory if non-existent and 
# the project is called "sample-service"
kn func create --runtime quarkus myfunc

# Create a function project that uses a CloudEvent based function signature
kn func create --template events myfunc
	`,
		SuggestFor: []string{"vreate", "creaet", "craete", "new"},
		PreRunE:    bindEnv("runtime", "template", "repositories", "confirm"),
	}

	cmd.Flags().BoolP("confirm", "c", false,
		"Prompt to confirm all configuration options (Env: $FUNC_CONFIRM)")
	cmd.Flags().StringP("runtime", "l", fn.DefaultRuntime,
		"Function runtime language/framework. Available runtimes: "+buildpacks.Runtimes()+" (Env: $FUNC_RUNTIME)")
	cmd.Flags().StringP("repositories", "r", filepath.Join(configPath(), "repositories"),
		"Path to extended template repositories (Env: $FUNC_REPOSITORIES)")
	cmd.Flags().StringP("template", "t", fn.DefaultTemplate,
		"Function template. Available templates: 'http' and 'events' (Env: $FUNC_TEMPLATE)")

	// Register tab-completeion function integration
	if err := cmd.RegisterFlagCompletionFunc("runtime", CompleteRuntimeList); err != nil {
		fmt.Println("internal: error while calling RegisterFlagCompletionFunc: ", err)
	}

	// The execution delegate is invoked with the command, arguments, and the
	// client creator.
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCreate(cmd, args, clientFn)
	}

	return cmd
}

func runCreate(cmd *cobra.Command, args []string, clientFn createClientFn) (err error) {
	config := newCreateConfig(args)

	if err = utils.ValidateFunctionName(config.Name); err != nil {
		return
	}

	if config, err = config.Prompt(); err != nil {
		if err == terminal.InterruptErr {
			return nil
		}
		return
	}

	function := fn.Function{
		Name:     config.Name,
		Root:     config.Path,
		Runtime:  config.Runtime,
		Template: config.Template,
	}

	client := clientFn(config.Repositories, config.Verbose)

	return client.Create(function)
}

type createConfig struct {
	// Name of the Function.
	Name string

	// Absolute path to Function on disk.
	Path string

	// Runtime language/framework.
	Runtime string

	// Repositories is an optional path that, if it exists, will be used as a source
	// for additional template repositories not included in the binary.  If not provided
	// explicitly as a flag (--repositories) or env (FUNC_REPOSITORIES), the default
	// location is $XDG_CONFIG_HOME/repositories ($HOME/.config/func/repositories)
	Repositories string

	// Template is the code written into the new Function project, including
	// an implementation adhering to one of the supported function signatures.
	// May also include additional configuration settings or examples.
	// For example, embedded are 'http' for a Function whose funciton signature
	// is invoked via straight HTTP requests, or 'events' for a Function which
	// will be invoked with CloudEvents.  These embedded templates contain a
	// minimum implementation of the signature itself and example tests.
	Template string

	// Verbose output
	Verbose bool

	// Confirm: confirm values arrived upon from environment plus flags plus defaults,
	// with interactive prompting (only applicable when attached to a TTY).
	Confirm bool
}

// newCreateConfig returns a config populated from the current execution context
// (args, flags and environment variables)
func newCreateConfig(args []string) createConfig {
	var path string
	if len(args) > 0 {
		path = args[0] // If explicitly provided, use.
	}

	derivedName, derivedPath := deriveNameAndAbsolutePathFromPath(path)
	return createConfig{
		Name:         derivedName,
		Path:         derivedPath,
		Repositories: viper.GetString("repositories"),
		Runtime:      viper.GetString("runtime"),
		Template:     viper.GetString("template"),
		Confirm:      viper.GetBool("confirm"),
		Verbose:      viper.GetBool("verbose"),
	}
}

// Prompt the user with value of config members, allowing for interaractive changes.
// Skipped if not in an interactive terminal (non-TTY), or if --confirm false (agree to
// all prompts) was set (default).
func (c createConfig) Prompt() (createConfig, error) {
	if !interactiveTerminal() || !c.Confirm {
		// Just print the basics if not confirming
		fmt.Printf("Project path: %v\n", c.Path)
		fmt.Printf("Function name: %v\n", c.Name)
		fmt.Printf("Runtime: %v\n", c.Runtime)
		fmt.Printf("Template: %v\n", c.Template)
		return c, nil
	}

	var qs = []*survey.Question{
		{
			Name: "path",
			Prompt: &survey.Input{
				Message: "Project path:",
				Default: c.Path,
			},
			Validate: func(val interface{}) error {
				derivedName, _ := deriveNameAndAbsolutePathFromPath(val.(string))
				return utils.ValidateFunctionName(derivedName)
			},
		},
		{
			Name: "runtime",
			Prompt: &survey.Select{
				Message: "Runtime:",
				Options: buildpacks.RuntimesList(),
				Default: c.Runtime,
			},
		},
		{
			Name: "template",
			Prompt: &survey.Input{
				Message: "Template:",
				Default: c.Template,
				// TODO add template suggestions: https://github.com/AlecAivazis/survey#suggestion-options
			},
		},
	}
	answers := struct {
		Template string
		Runtime  string
		Path     string
	}{}
	err := survey.Ask(qs, &answers)
	if err != nil {
		return createConfig{}, err
	}

	derivedName, derivedPath := deriveNameAndAbsolutePathFromPath(answers.Path)

	return createConfig{
		Name:     derivedName,
		Path:     derivedPath,
		Runtime:  answers.Runtime,
		Template: answers.Template,
	}, nil
}
