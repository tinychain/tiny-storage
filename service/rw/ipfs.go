package rw

import (
	"errors"
	"github.com/tinychain/tiny-storage/types"
	"os"
	"path"
	"time"
)

var (
	dataDir     = ".data"
	storePrefix = "ipfs_"

	errSizeMismatch = errors.New("size in IPFS is larger than that in fileDesc")
	errDataTimeout  = errors.New("data reaches timeout")
	errDataNotFound = errors.New("data not found")
)

// AddDataToIPFS add data with a given path to IPFS network.
func (rw *RWService) AddDataToIPFS(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	cid, err := rw.ipfs.Add(file)
	if err != nil {
		return "", err
	}
	return cid, nil
}

// Fetch fetches data from IPFS, and save in local directory.
// If timeout reaches, it's treated as data not found.
func (rw *RWService) GetFromIPFS(cid string) (os.FileInfo, error) {
	resChan := make(chan error)
	timer := time.NewTimer(rw.conf.fetchTimeout)
	go func() {
		err := rw.ipfs.Get(cid, path.Join(dataDir, storePrefix+cid))
		select {
		case resChan <- err:
		default:
			rw.DeleteData(cid) // delete unused data from IPFS
			return
		}
	}()

	select {
	case err := <-resChan:
		if err != nil {
			rw.log.Errorf("failed to fetch data from IPFS, %s", err)
		}
		return rw.GetFromLocal(cid)
	case <-timer.C:
		resChan = nil
		rw.log.Warningf("fetch data with cid %s timeout", cid)
		return nil, errDataNotFound
	}

}

// Get returns the file name of data with given cid. If not exist in local,
// the node will retrieves data from IPFS, and save in local directory.
func (rw *RWService) GetFromLocal(cid string) (os.FileInfo, error) {
	fname := path.Join(dataDir, rw.constructPrefix(cid))
	file, err := os.Open(fname)
	if err == nil {
		fi, err := file.Stat()
		if err != nil {
			return nil, err
		}
		return fi, err
	}

	return nil, err
}

func (rw *RWService) DeleteData(cid string) error {
	fname := path.Join(dataDir, rw.constructPrefix(cid))
	if err := os.Remove(fname); err != nil {
		return err
	}
	return nil
}

func (rw *RWService) VerifyWithIPFS(fd *types.FileDesc) error {
	// Check is there temporary data in local
	fi, err := rw.GetFromLocal(fd.Cid)
	if err == nil {
		if fi.Size() >= int64(fd.Size) {
			return errSizeMismatch
		}

		if rw.checkTimeout(fi.ModTime(), fd.Duration) {
			rw.DeleteData(fd.Cid)
			return errDataTimeout
		}
	}

	fi, err = rw.GetFromIPFS(fd.Cid)
	if err == nil {
		if fi.Size() >= int64(fd.Size) {
			return errSizeMismatch
		}

		if rw.checkTimeout(fi.ModTime(), fd.Duration) {
			rw.DeleteData(fd.Cid)
			return errDataTimeout
		}
	}

	return err
}

func (rw *RWService) checkTimeout(createTime time.Time, duration time.Duration) bool {
	diff := time.Now().Sub(createTime)
	return diff >= duration
}

func (rw *RWService) constructPrefix(cid string) string {
	return storePrefix + cid
}
