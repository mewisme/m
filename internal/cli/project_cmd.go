package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/workspace"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Inspect the current project",
	}
	cmd.AddCommand(newProjectInfoCmd())
	return cmd
}

func newProjectInfoCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Print project root, identity, and workspace members",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "project.info", "", "missing app context")
			}
			p, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			var members []string
			var patterns []string
			if p.Doc != nil {
				patterns, err = p.Doc.WorkspacePatterns()
				if err != nil {
					return err
				}
				if len(patterns) > 0 {
					idx, err := workspace.BuildIndex(p.Root)
					if err != nil {
						return err
					}
					members = idx.Members
					patterns = idx.Patterns
				}
			}
			name, version := "", ""
			if p.Doc != nil {
				name, version = p.Doc.Name, p.Doc.Version
			}
			if asJSON {
				doc := map[string]any{
					"root":       p.Root,
					"identity":   string(p.Identity),
					"name":       name,
					"version":    version,
					"workspaces": patterns,
					"members":    members,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(doc)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "root=%s\n", p.Root)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "identity=%s\n", p.Identity)
			if name != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "name=%s\n", name)
			}
			if version != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "version=%s\n", version)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "members=%d\n", len(members))
			for _, m := range members {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "member=%s\n", m)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return cmd
}

func newPkgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Read package.json fields",
	}
	cmd.AddCommand(newPkgGetCmd())
	return cmd
}

func newPkgGetCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "get <field>",
		Short: "Get a package.json field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "pkg.get", "", "missing app context")
			}
			p, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			field := args[0]
			val, err := pkgField(p, field)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{"field": field, "value": val})
			}
			switch v := val.(type) {
			case string:
				_, err = fmt.Fprintln(cmd.OutOrStdout(), v)
			case nil:
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "")
			default:
				b, e := json.Marshal(v)
				if e != nil {
					return apperr.Wrap(apperr.Manifest, "pkg.get", field, e)
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return cmd
}

func pkgField(p *project.Project, field string) (any, error) {
	if p == nil || p.Doc == nil {
		return nil, apperr.New(apperr.Manifest, "pkg.get", field, "no package.json loaded")
	}
	d := p.Doc
	switch field {
	case "name":
		return d.Name, nil
	case "version":
		return d.Version, nil
	case "private":
		return d.Private, nil
	case "packageManager":
		return d.PackageManager, nil
	case "dependencies":
		return d.Dependencies, nil
	case "devDependencies":
		return d.DevDependencies, nil
	case "optionalDependencies":
		return d.OptionalDependencies, nil
	case "peerDependencies":
		return d.PeerDependencies, nil
	case "scripts":
		return d.Scripts, nil
	case "engines":
		return d.Engines, nil
	case "workspaces":
		if len(d.Workspaces) == 0 {
			return nil, nil
		}
		var v any
		if err := json.Unmarshal(d.Workspaces, &v); err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "pkg.get", field, err)
		}
		return v, nil
	case "bin":
		if len(d.Bin) == 0 {
			return nil, nil
		}
		var v any
		if err := json.Unmarshal(d.Bin, &v); err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "pkg.get", field, err)
		}
		return v, nil
	default:
		return nil, apperr.New(apperr.Usage, "pkg.get", field, "unknown field")
	}
}
