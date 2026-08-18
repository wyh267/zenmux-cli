// Package client 的文件辅助函数。

package client

import "os"

// openFile 创建/截断目标文件用于写入。
func openFile(dest string) (*os.File, error) {
	return os.Create(dest)
}
