package path

import (
	"fmt"
	"strconv"
)

var configPath string = "./storage/config"
var txPoolPath string = "./storage/pool/"
var blockPath string = "./storage/block/"

func SetConfigPath(_path string) {
	configPath = _path
}

func SetTxPoolPath(_path string) {
	txPoolPath = _path
}

func SetBlockDirPath(_path string) {
	blockPath = _path
}

func GetConfigPath() string {
	return configPath
}

func GetTxPoolPath() string {
	return txPoolPath
}

func GetBlockDirPath() string {
	return blockPath
}

func GetBlockPath(h int) string {
	return blockPath + strconv.Itoa(h)
}

func GetPoolTxPath(hash []byte) string {
	return txPoolPath + fmt.Sprintf("%x", hash)
}
