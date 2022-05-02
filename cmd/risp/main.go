// Package main is the RISP application entrypoint.
package main

import (
	"context"
	"fmt"
	"strconv"

	"risp/internal"
	"risp/internal/app/apps"
	"risp/internal/app/cfg"
	"risp/internal/pkg/log"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// CLI command definitions.
var (
	logger logrus.FieldLogger = logrus.StandardLogger()

	rootCmd = &cobra.Command{
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	}

	clientCmd = &cobra.Command{
		Use:   "client [sequence_length]",
		Short: "Starts a RISP client.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 1 {
				return nil
			}
			_, err := strconv.ParseUint(args[0], 10, 16)
			if err != nil {
				return errors.Wrap(err, "parse sequence length argument failed")
			}
			return nil
		},
		RunE: runCmd,
	}

	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Starts a RISP server.",
		RunE:  runCmd,
	}
)

func newApp(_ context.Context, cmd *cobra.Command, args []string) (apps.App, []string, error) {
	var err error
	var app apps.App
	switch cmd.Name() {
	case "client":
		app, err = apps.NewClientApp(cfg.PortFromEnv())
		if err != nil {
			return nil, nil, errors.Wrap(err, "new client app failed")
		}
		args = append([]string{cmd.Name()}, args[1:]...)
		return app, args, nil
	case "server":
		app, err = apps.NewServerApp(cfg.PortFromEnv())
		if err != nil {
			return nil, nil, errors.Wrap(err, "new server app failed")
		}
		args = append([]string{cmd.Name()}, args[1:]...)
		return app, args, nil
	default:
		return nil, nil, fmt.Errorf("unknown command: %s", cmd.Name())
	}
}

func runCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	if err := chainedCheck(
		ctx,
		envCheck,
	); err != nil {
		return errors.Wrap(err, "chained check failed")
	}
	app, args, err := newApp(cmd.Context(), cmd, args)
	if err != nil {
		return errors.Wrapf(err, "new %s app failed", cmd.Name())
	}
	return errors.Wrap(app.Run(ctx, args), "run app failed")
}

func envCheck(ctx context.Context) error {
	err := internal.ValidateEnv()
	if err != nil {
		return errors.Wrap(err, "validate env failed")
	}
	log.SetLogger(internal.LogLevel)
	return nil
}

func chainedCheck(ctx context.Context, checks ...func(context.Context) error) error {
	for _, check := range checks {
		err := check(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func init() {
	err := internal.RegisterCommandFlags(rootCmd, []*internal.Flag{
		&internal.EnvFlag,
		&internal.LogLevelFlag,

		&internal.HealthPortFlag,
		&internal.PortFlag,

		&internal.MaxGoroutinesFlag,
	})
	if err != nil {
		logger.Fatalln(err)
	}

	err = internal.RegisterCommandFlags(clientCmd, []*internal.Flag{
		&internal.ClientTickerMSFlag,
		&internal.ClientKillswitchMSFlag,
	})
	if err != nil {
		logger.Fatalln(err)
	}

	err = internal.RegisterCommandFlags(serverCmd, []*internal.Flag{
		&internal.ServerTickerMSFlag,
	})
	if err != nil {
		logger.Fatalln(err)
	}

	rootCmd.AddCommand(
		clientCmd,
		serverCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(errors.Wrap(err, "execute root command failed"))
	}
}
