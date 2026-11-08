// Package ipregion 负责桌面端 ipregion.db 的路径、换文件，以及目标是否落在指定地区。
// 打开和查询走 engine。
package ipregion

import (
	"desktop/internal/storage"
	"errors"
	"os"
	"path/filepath"

	"github.com/penndev/prism/engine"
)

const fileName = "ipregion.db"

// Path 返回配置目录下的 ipregion.db。
func Path() (string, error) {
	dir, err := storage.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Ensure 文件存在且当前没打开时，用 engine 打开固定路径。
func Ensure() error {
	if engine.IpregionStatus() != nil {
		return nil
	}
	path, err := Path()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return engine.SetIpregionDB(path, os.TempDir())
}

// Close 关掉 engine 里的库。
func Close() {
	engine.CloseIpregionDB()
}

// Replace 用已经校验过的 tmp 覆盖固定路径上的库，并用 engine 重新打开。
func Replace(tmp string) error {
	path, err := Path()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	Close()
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return engine.SetIpregionDB(path, os.TempDir())
}

// Remove 关掉库并删除固定路径上的文件。
func Remove() error {
	path, err := Path()
	if err != nil {
		return err
	}
	Close()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Read 打开固定路径上的库，返回 engine 的摘要。没有库或打不开时返回 nil。
func Read() *engine.DBStatus {
	if err := Ensure(); err != nil {
		return nil
	}
	return engine.IpregionStatus()
}

// Contains 报告 address 是否落在 areaIDs 任一地区（含其下级）。
// 库打不开或查不到地区时返回 false。
func Contains(address string, areaIDs []uint32) bool {
	if len(areaIDs) == 0 {
		return false
	}
	if err := Ensure(); err != nil {
		return false
	}
	list := engine.Lookup(address)
	if list == nil || list.Len() == 0 {
		return false
	}
	set := make(map[int64]struct{}, len(areaIDs))
	for _, id := range areaIDs {
		if id != 0 {
			set[int64(id)] = struct{}{}
		}
	}
	for i := 0; i < list.Len(); i++ {
		if _, ok := set[list.Get(i)]; ok {
			return true
		}
	}
	return false
}
