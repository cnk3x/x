package fsx

import (
	"fmt"
	"os"
)

func NotExist(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.ErrExist
}

func FileExist(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return os.ErrExist
	}
	return nil
}

// FileCheck 存在是非空文件：跳过，存在空文件：删除，不存在：正常返回，打开出错：报错
func FileCheck(target string) (skipExist bool, err error) {
	//检查并跳过存在
	stat, err := os.Stat(target)

	// 路径打开失败
	if err != nil {
		if os.IsNotExist(err) { //不存在，正常反问
			return false, nil
		}
		return false, fmt.Errorf("路径打开失败： %w", err)
	}

	//是文件
	if stat.Mode().IsRegular() {
		if stat.Size() == 0 {
			return false, os.Remove(target) //空文件，删除
		}
		return true, nil //文件非空, 存在跳过
	}

	//存在非文件
	return false, fmt.Errorf("路径冲突： %w", err)
}
