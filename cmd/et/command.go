package main

import "github.com/spf13/cobra"

var Create = &cobra.Command{
	Use:   "go",
	Short: "You can created Microservice.",
	Long:  "Template project to create microservice include required folders and basic files.",
	Run: func(cmd *cobra.Command, args []string) {
		// PrompCreate()
	},
}
