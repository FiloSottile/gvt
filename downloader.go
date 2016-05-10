package main

import (
	"strings"
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
	wcsMu sync.Mutex
	wcs   map[cacheKey]*cacheEntry

	reposMu sync.RWMutex
	repos   map[string]vendor.RemoteRepo
	reposI  map[string]vendor.RemoteRepo
}

var GlobalDownloader = Downloader{}

func init() {
	GlobalDownloader.wcs = make(map[cacheKey]*cacheEntry)
	GlobalDownloader.repos = make(map[string]vendor.RemoteRepo)
	GlobalDownloader.reposI = make(map[string]vendor.RemoteRepo)
}

// Get returns a cached WorkingCopy, or runs RemoteRepo.Checkout
func (d *Downloader) Get(repo vendor.RemoteRepo, branch, tag, revision string) (vendor.WorkingCopy, error) {
	key := cacheKey{
		url: repo.URL(), repoType: repo.Type(),
		branch: branch, tag: tag, revision: revision,
	}
	d.wcsMu.Lock()
	if entry, ok := d.wcs[key]; ok {
		d.wcsMu.Unlock()
		entry.wg.Wait()
		return entry.v, entry.err
	}

	entry := &cacheEntry{}
	entry.wg.Add(1)
	d.wcs[key] = entry
	d.wcsMu.Unlock()

	entry.v, entry.err = repo.Checkout(branch, tag, revision)
	entry.wg.Done()
	return entry.v, entry.err
}

func (d *Downloader) Flush() error {
	d.wcsMu.Lock()
	defer d.wcsMu.Unlock()

	for _, entry := range d.wcs {
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

// DeduceRemoteRepo is a cached version of vendor.DeduceRemoteRepo
func (d *Downloader) DeduceRemoteRepo(path string, insecure bool) (vendor.RemoteRepo, string, error) {
	cache := d.repos
	if insecure {
		cache = d.reposI
	}

	d.reposMu.RLock()
	for p, repo := range cache {
		if path == p || strings.HasPrefix(path, p+"/") {
			d.reposMu.RUnlock()
			extra := strings.Trim(strings.TrimPrefix(path, p), "/")
			return repo, extra, nil
		}
	}
	d.reposMu.RUnlock()

	repo, extra, err := vendor.DeduceRemoteRepo(path, insecure)
	if err != nil {
		return repo, extra, err
	}

	if !strings.HasSuffix(path, extra) {
		// Shouldn't happen, but in case just bypass the cache
		return repo, extra, err
	}
	basePath := strings.Trim(strings.TrimSuffix(path, extra), "/")
	d.reposMu.Lock()
	cache[basePath] = repo
	d.reposMu.Unlock()

	return repo, extra, err
}
