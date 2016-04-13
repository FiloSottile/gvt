package main

import (
	"sync"

	"github.com/FiloSottile/gvt/gbvendor"
)

type cacheKey struct {
	url, repoType         string
	branch, tag, revision string
}

type cacheEntry struct {
	wg  sync.WaitGroup
	v   vendor.WorkingCopy
	err error
}

// Downloader acts as a cache for downloaded repositories
type Downloader struct {
	mu sync.Mutex
	m  map[cacheKey]*cacheEntry
}

var GlobalDownloader = Downloader{}

func init() {
	GlobalDownloader.m = make(map[cacheKey]*cacheEntry)
}

// Get returns a cached WorkingCopy, or runs RemoteRepo.Checkout
func (d *Downloader) Get(repo vendor.RemoteRepo, branch, tag, revision string) (vendor.WorkingCopy, error) {
	key := cacheKey{
		url: repo.URL(), repoType: repo.Type(),
		branch: branch, tag: tag, revision: revision,
	}
	d.mu.Lock()
	if entry, ok := d.m[key]; ok {
		d.mu.Unlock()
		entry.wg.Wait()
		return entry.v, entry.err
	}

	entry := &cacheEntry{}
	entry.wg.Add(1)
	d.m[key] = entry
	d.mu.Unlock()

	entry.v, entry.err = repo.Checkout(branch, tag, revision)
	entry.wg.Done()
	return entry.v, entry.err
}

func (d *Downloader) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, entry := range d.m {
		entry.wg.Wait()
		if entry.err != nil {
			continue
		}
		if err := entry.v.Destroy(); err != nil {
			return err
		}
	}
	return nil
}
