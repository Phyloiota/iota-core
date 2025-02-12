package prunable

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/runtime/ioutils"
	iotago "github.com/iotaledger/iota.go/v4"
)

// indexToRealm converts an baseIndex to a realm with some shifting magic.
func indexToRealm(index iotago.SlotIndex) kvstore.Realm {
	return []byte{
		byte(0xff & index),
		byte(0xff & (index >> 8)),
		byte(0xff & (index >> 16)),
		byte(0xff & (index >> 24)),
		byte(0xff & (index >> 32)),
		byte(0xff & (index >> 40)),
		byte(0xff & (index >> 48)),
		byte(0xff & (index >> 54)),
	}
}

func dbPathFromIndex(base string, index iotago.SlotIndex) string {
	return filepath.Join(base, strconv.FormatInt(int64(index), 10))
}

type dbInstanceFileInfo struct {
	baseIndex iotago.SlotIndex
	path      string
}

// getSortedDBInstancesFromDisk returns an ASC sorted list of db instances from the given base directory.
func getSortedDBInstancesFromDisk(baseDir string) (dbInfos []*dbInstanceFileInfo) {
	files, err := os.ReadDir(baseDir)
	if err != nil {
		panic(err)
	}

	files = lo.Filter(files, func(e os.DirEntry) bool { return e.IsDir() })
	dbInfos = lo.Map(files, func(e os.DirEntry) *dbInstanceFileInfo {
		atoi, convErr := strconv.Atoi(e.Name())
		if convErr != nil {
			return nil
		}

		return &dbInstanceFileInfo{
			baseIndex: iotago.SlotIndex(atoi),
			path:      filepath.Join(baseDir, e.Name()),
		}
	})
	dbInfos = lo.Filter(dbInfos, func(info *dbInstanceFileInfo) bool { return info != nil })

	sort.Slice(dbInfos, func(i, j int) bool {
		return dbInfos[i].baseIndex < dbInfos[j].baseIndex
	})

	return dbInfos
}

func dbPrunableDirectorySize(base string, index iotago.SlotIndex) (int64, error) {
	return ioutils.FolderSize(dbPathFromIndex(base, index))
}
