package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	gateway "icooclaw.gateway"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "启动网关",
	Long: `启动网关

	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGateway()
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
}

func runGateway() error {
	// 启动网关
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 启动网关
	g, err := gateway.NewRESTGateway()
	if err != nil {
		return fmt.Errorf("网关启动失败: %w", err)
	}

	g.Start(ctx)

	// 等待上下文取消
	<-ctx.Done()

	return nil
}
