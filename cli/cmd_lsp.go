package main

import (
	"fmt"

	"github.com/spf13/cobra"
)
import "github.com/theirish81/fml/lsp"

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Run an LSP server for the FML language",
	Run: func(cmd *cobra.Command, args []string) {
		if port == 0 {
			port = 8080
		}
		termChan := make(chan bool, 1)
		if tcp {
			go func() {
				if err := lsp.RunTCP(cmd.Context(), fmt.Sprintf(":%d", port)); err != nil {
					fmt.Println(err)
				}
				termChan <- true
			}()
		}
		if ws {
			go func() {
				if err := lsp.RunWS(cmd.Context(), fmt.Sprintf(":%d", port)); err != nil {
					fmt.Println(err)
				}
				termChan <- true
			}()
		}
		if stdio {
			go func() {
				if err := lsp.RunStdio(cmd.Context()); err != nil {
					cmd.PrintErrln(err)
				}
				termChan <- true
			}()
		}
		if ws || tcp || stdio {
			<-termChan
		}
	},
}

func init() {
	lspCmd.Flags().BoolVarP(&stdio, "stdio", "", true, "Standard IO mode")
	lspCmd.Flags().BoolVarP(&ws, "websocket", "", false, "Websocket mode")
	lspCmd.Flags().BoolVarP(&tcp, "tcp", "", false, "TCP mode")
	lspCmd.Flags().IntVarP(&port, "port", "p", 8080, "The port to run the LSPs on")
}
