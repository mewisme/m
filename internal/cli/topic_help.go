package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/help"
	"github.com/mewisme/mew/internal/presentation"
	helpmd "github.com/mewisme/mew/internal/presentation/help"
	"github.com/mewisme/mew/internal/presentation/pager"
)

// configureTopicHelp replaces Cobra's default help command with command+topic routing.
func configureTopicHelp(root *cobra.Command) {
	var pagerFlag string
	helpCmd := &cobra.Command{
		Use:   "help [command|topic] [code]",
		Short: "Help about any command or documentation topic",
		Long: `Help about commands and curated terminal topics.

Command names win when a name is both a command and a topic id.
Topic examples: m help runner, m help errors ERR_M_LOCKFILE.
Ordinary command help stays concise; topic help may use an optional pager.`,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     topicHelpCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHelpCommand(cmd, args, pagerFlag)
		},
	}
	helpCmd.Flags().StringVar(&pagerFlag, "pager", "auto", "topic pager: auto|always|never")
	root.SetHelpCommand(helpCmd)
}

func runHelpCommand(cmd *cobra.Command, args []string, pagerFlag string) error {
	root := cmd.Root()
	if len(args) == 0 {
		return root.Help()
	}

	// Command path wins over topic id when both exist.
	if target, ok := findHelpCommand(root, args); ok {
		return target.Help()
	}

	reg, err := help.Default()
	if err != nil {
		return err
	}
	topic, body, err := reg.ResolveArgs(args)
	if err != nil {
		msg := err.Error()
		if ae, ok := err.(*apperr.Error); ok && ae.Message != "" {
			msg = ae.Message
		}
		return apperr.New(apperr.Usage, "help", strings.Join(args, " "),
			msg+"\n\n"+reg.FormatTopicList()+"\n\nUse \"m <command> --help\" for command help.")
	}
	if topic == nil {
		return root.Help()
	}
	return writeTopicHelp(cmd, body, pagerFlag)
}

func findHelpCommand(root *cobra.Command, args []string) (*cobra.Command, bool) {
	if root == nil || len(args) == 0 {
		return nil, false
	}
	target, rest, err := root.Find(args)
	if err != nil || target == nil || target == root {
		return nil, false
	}
	if target.Name() == "help" {
		return nil, false
	}
	// Exact command path only — leftover args mean this is not a command help target.
	if len(rest) != 0 {
		return nil, false
	}
	return target, true
}

func writeTopicHelp(cmd *cobra.Command, body []byte, pagerFlag string) error {
	g := ownerFlags(cmd.Root())
	var cfg *config.Effective
	if ac := app.FromContext(cmd.Context()); ac != nil {
		cfg = ac.Config
	}
	ctrl, err := g.controller(cmd, cfg)
	if err != nil {
		return wrapPresentationErr(err)
	}
	caps := ctrl.Capabilities()
	opts := ctrl.Options()
	eff := presentation.Effective(opts, caps)

	structured := opts.Structured()
	plain := structured || opts.EffectiveOutput == presentation.OutputPlain ||
		!caps.StdoutTTY || caps.CI || caps.DumbTerminal || caps.NoColorEnv ||
		opts.Color == presentation.TriNever || eff.Accessible || !eff.UseColor
	human := !structured
	style := "dark"
	if caps.Background == presentation.BackgroundLight {
		style = "light"
	}

	rendered, err := helpmd.Render(string(body), helpmd.RenderOptions{
		Width:      eff.Width,
		Plain:      plain,
		Accessible: eff.Accessible,
		Hyperlinks: caps.Hyperlinks && !plain && !eff.Accessible,
		Style:      style,
	})
	if err != nil {
		return err
	}

	configPager := ""
	if cfg != nil {
		configPager = config.String(cfg, "ui.pager", "")
	}
	plan, err := pager.Resolve(pager.Input{
		Flag:        pagerFlag,
		MEWPager:    os.Getenv("MEW_PAGER"),
		ConfigPager: configPager,
		PAGER:       os.Getenv("PAGER"),
		StdoutTTY:   caps.StdoutTTY,
		Human:       human,
		CI:          caps.CI,
		Accessible:  eff.Accessible,
		LineCount:   pager.LineCount(rendered),
	})
	if err != nil {
		return err
	}
	return pager.WritePaged(cmd.Context(), cmd.OutOrStdout(), rendered, plan)
}

func topicHelpCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_ = toComplete
	if len(args) >= 2 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	reg, err := help.Default()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	if len(args) == 1 && args[0] == "errors" {
		for _, t := range reg.Topics() {
			if strings.HasPrefix(t.ID, "errors/ERR_M_") {
				out = append(out, strings.TrimPrefix(t.ID, "errors/"))
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	for _, c := range cmd.Root().Commands() {
		if !c.IsAvailableCommand() || c.Hidden || c.Name() == "help" {
			continue
		}
		out = append(out, c.Name())
	}
	for _, t := range reg.Topics() {
		if strings.HasPrefix(t.ID, "errors/") {
			continue
		}
		out = append(out, t.ID)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
