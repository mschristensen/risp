// Package internal contains the applciation configuration.
// By default, configuration is read from the envionment variables,
// with sensible defaults in place.
// These values can be overridden with command-line flags.
package internal

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// LocalEnv is a string indicating that the application is running a local environment.
	LocalEnv = "local"
	// TestEnv is a string indicating that the application is running a test environment.
	TestEnv = "test"
	// DevEnv is a string indicating that the application is running a development environment.
	DevEnv = "dev"
	// ProdEnv is a string indicating that the application is running a production environment.
	ProdEnv = "prod"
)

// Flag describes a piece of application configuration.
type Flag struct {
	Name         string
	Usage        string
	defaultValue interface{}
	Value        interface{}
}

// Application configuration flags.
var (
	EnvFlag = Flag{
		Name:  "env",
		Usage: "Describes the current environment and should be one of: local, test, dev, prod.",
		Value: &Env,
	}
	LogLevelFlag = Flag{
		Name:  "log_level",
		Usage: "Sets the log level and should be one of: debug, info, warn, error.",
		Value: &LogLevel,
	}

	HealthPortFlag = Flag{
		Name:  "health_port",
		Usage: "The port the health server should listen on.",
		Value: &HealthPort,
	}
	PortFlag = Flag{
		Name:  "port",
		Usage: "The port the gRPC server should listen on.",
		Value: &Port,
	}

	MaxGoroutinesFlag = Flag{
		Name:  "max_goroutines",
		Usage: "The maximum allowed number of goroutines that can be spawned before healthchecks fail.",
		Value: &MaxGoroutines,
	}
)

// Application configuration variables.
var (
	Env      string
	LogLevel string

	HealthPort int
	Port       int

	MaxGoroutines int
)

// setDefault sets the default value of the flag to the given value iff
// it is not already provided by the environment.
func setDefault(flag *Flag, value interface{}) {
	switch v := value.(type) {
	case string:
		viper.SetDefault(flag.Name, v)
		flag.defaultValue = viper.GetString(flag.Name)
		valueVar := flag.Value.(*string)
		*valueVar = viper.GetString(flag.Name)
	case []string:
		viper.SetDefault(flag.Name, v)
		flag.defaultValue = viper.GetStringSlice(flag.Name)
		valueVar := flag.Value.(*[]string)
		*valueVar = viper.GetStringSlice(flag.Name)
	case int:
		viper.SetDefault(flag.Name, v)
		flag.defaultValue = viper.GetInt(flag.Name)
		valueVar := flag.Value.(*int)
		*valueVar = viper.GetInt(flag.Name)
	case bool:
		viper.SetDefault(flag.Name, v)
		flag.defaultValue = viper.GetBool(flag.Name)
		valueVar := flag.Value.(*bool)
		*valueVar = viper.GetBool(flag.Name)
	default:
		panic(errors.Wrap(fmt.Errorf("unsupported flag type %T for flag %s", v, spew.Sdump(flag)), "set default failed"))
	}
}

func init() {
	viper.AutomaticEnv()

	setDefault(&EnvFlag, "local")
	setDefault(&LogLevelFlag, "debug")

	setDefault(&HealthPortFlag, 8080)
	setDefault(&PortFlag, 8081)

	setDefault(&MaxGoroutinesFlag, 200)
}

// RegisterCommandFlags registers the given flags with cobra.
func RegisterCommandFlags(cmd *cobra.Command, flags []*Flag) error {
	for _, flag := range flags {
		switch defaultVal := flag.defaultValue.(type) {
		case string:
			val := flag.Value.(*string)
			cmd.PersistentFlags().StringVar(val, flag.Name, defaultVal, flag.Usage)
		case []string:
			val := flag.Value.(*[]string)
			cmd.PersistentFlags().StringSliceVar(val, flag.Name, defaultVal, flag.Usage)
		case int:
			val := flag.Value.(*int)
			cmd.PersistentFlags().IntVar(val, flag.Name, defaultVal, flag.Usage)
		case bool:
			val := flag.Value.(*bool)
			cmd.PersistentFlags().BoolVar(val, flag.Name, defaultVal, flag.Usage)
		default:
			return fmt.Errorf("unsupported flag type %T for flag %s", defaultVal, spew.Sdump(flag))
		}
	}
	return nil
}

func normaliseEnvString(env string) (string, error) {
	normalised := strings.ToLower(env)
	// permit long form spellings
	if normalised == "Development" {
		normalised = DevEnv
	} else if normalised == "Production" {
		normalised = ProdEnv
	}
	// ensure env is a valid value
	if normalised != TestEnv && normalised != LocalEnv && normalised != DevEnv && normalised != ProdEnv {
		return "", errors.New("Invalid environment: " + normalised)
	}
	return normalised, nil
}

// ValidateEnv ensures the environment is valid, fixing any problems where possible,
// and returns any error encountered.
func ValidateEnv() error {
	norm, err := normaliseEnvString(Env)
	if err != nil {
		return errors.Wrap(err, "normalise env string failed")
	}
	viper.Set(Env, norm)
	if err != nil {
		return err
	}
	return nil
}
