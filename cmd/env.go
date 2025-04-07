package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/ciclebyte/aiExplain/assets"
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "env",
	Long:  `generate .env file for the mysql and ai config`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查当前目录是否存在.env文件
		if _, err := os.Stat(".env"); err == nil {
			fmt.Println("当前目录已存在.env文件")
			return
		}

		// 从嵌入资源读取模板文件
		envContent, err := fs.ReadFile(assets.Resources, "resources/.env")
		if err != nil {
			fmt.Printf("读取模板文件失败: %v\n", err)
			return
		}

		// 写入.env文件
		if err := os.WriteFile(".env", envContent, 0644); err != nil {
			fmt.Printf("写入.env文件失败: %v\n", err)
			return
		}

		fmt.Println("已成功生成.env文件")
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
