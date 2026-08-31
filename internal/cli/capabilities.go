package cli

import (
	"fmt"

	"github.com/QYVORA/qyvora-jabari/internal/capabilities"
	"github.com/spf13/cobra"
)

func newCapabilitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "Show native capabilities and optional integrations",
		Long: `Display Jabari's native capabilities versus optional external tool integrations.

Native capabilities are built into Jabari and require no external dependencies.
Integrations are optional tools that extend Jabari's functionality.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			byCategory := capabilities.ByCategory()

			// Define category order
			categories := []capabilities.Category{
				capabilities.CategoryAndroidEngine,
				capabilities.CategoryAPKEngine,
				capabilities.CategoryAnalysis,
				capabilities.CategoryIntegrations,
			}

			for _, cat := range categories {
				caps := byCategory[cat]
				if len(caps) == 0 {
					continue
				}

				fmt.Printf("\n%s\n", cat)

				for _, cap := range caps {
					symbol := ""
					switch cap.Status {
					case capabilities.StatusNative:
						symbol = "✓"
					case capabilities.StatusIntegration:
						symbol = "○"
					case capabilities.StatusMissing:
						symbol = "✗"
					}

					fmt.Printf(" %s %-30s %s\n", symbol, cap.Name, cap.Description)
				}
			}

			// Summary
			native, integration, missing := capabilities.Summary()
			fmt.Printf("\n")
			fmt.Printf("Native: %d  |  Integrations: %d  |  Missing: %d\n", native, integration, missing)
			fmt.Printf("\n✓ = native  ○ = available integration  ✗ = missing integration\n")

			return nil
		},
	}
}

// cmdCapabilities shows capabilities in the interactive console
func (c *jabariConsole) cmdCapabilities() {
	byCategory := capabilities.ByCategory()

	categories := []capabilities.Category{
		capabilities.CategoryAndroidEngine,
		capabilities.CategoryAPKEngine,
		capabilities.CategoryAnalysis,
		capabilities.CategoryIntegrations,
	}

	for _, cat := range categories {
		caps := byCategory[cat]
		if len(caps) == 0 {
			continue
		}

		c.ui.Section(string(cat))

		rows := [][]string{}
		for _, cap := range caps {
			status := ""
			switch cap.Status {
			case capabilities.StatusNative:
				status = c.ui.Green("✓ native")
			case capabilities.StatusIntegration:
				status = c.ui.Green("○ integration")
			case capabilities.StatusMissing:
				status = c.ui.Amber("✗ missing")
			}

			rows = append(rows, []string{cap.Name, status, cap.Description})
		}

		c.ui.Table([]string{"capability", "status", "description"}, rows)
	}

	native, integration, missing := capabilities.Summary()
	c.ui.Status("summary", "Native: %d | Integrations: %d | Missing: %d",
		native, integration, missing)
}
