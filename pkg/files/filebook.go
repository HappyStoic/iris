package files

import (
	"bufio"
	"io/ioutil"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/org"
)

type FileMeta struct {
	ExpiredAt time.Time
	Expired   bool

	Available bool
	Path      string

	Rights      []*org.Org
	Severity    Severity
	Description interface{}
}

func GetFileCid(path string) (*cid.Cid, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	reader := bufio.NewReader(f)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return GetBytesCid(data)
}

func GetBytesCid(data []byte) (*cid.Cid, error) {
	var builder cid.V0Builder
	c, err := cid.V0Builder.Sum(builder, data)
	return &c, err
}

type FileBook struct {
	files map[cid.Cid]*FileMeta
}

func NewFileBook() *FileBook {
	return &FileBook{files: make(map[cid.Cid]*FileMeta)}
}

func (fb *FileBook) Get(cid *cid.Cid) *FileMeta {
	if meta, exists := fb.files[*cid]; exists {
		return meta
	}
	return nil
}

func (fb *FileBook) AddFile(cid *cid.Cid, meta *FileMeta) error {
	if _, exists := fb.files[*cid]; exists {
		return errors.Errorf("file with cid %s already exists", cid.String())
	}
	fb.files[*cid] = meta
	return nil
}
