package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// writeStaticOut prints presentation-rendered human output to stdout.
func writeStaticOut(cmd *cobra.Command, text string) error {
	if text == "" {
		return nil
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

// writeStaticErr prints presentation-rendered human output to stderr.
func writeStaticErr(cmd *cobra.Command, text string) error {
	if text == "" {
		return nil
	}
	_, err := fmt.Fprintln(cmd.ErrOrStderr(), text)
	return err
}

// writeStaticPrint prints presentation-rendered output without a trailing newline.
func writeStaticPrint(cmd *cobra.Command, text string) error {
	if text == "" {
		return nil
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), text)
	return err
}
