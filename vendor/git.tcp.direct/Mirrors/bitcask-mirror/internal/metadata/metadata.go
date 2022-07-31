package metadata

import (
	"os"

	"git.tcp.direct/Mirrors/bitcask-mirror/internal"
)

type MetaData struct {
	IndexUpToDate    bool  `json:"index_up_to_date"`
	ReclaimableSpace int64 `json:"reclaimable_space"`
}

func (m *MetaData) Save(path string, mode os.FileMode) error {
	return internal.SaveJsonToFile(m, path, mode)
}

func Load(path string) (*MetaData, error) {
	var m MetaData
	err := internal.LoadFromJsonFile(path, &m)
	return &m, err
}
