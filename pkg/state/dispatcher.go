package state

import (
	"sync"

	"github.com/nenad/couch/pkg/media"
	"github.com/nenad/couch/pkg/storage"
)

type (
	ScrapeResult struct {
		Value []storage.Magnet
		Error error
	}

	ExtractResult struct {
		Value []storage.Download
		Error error
	}

	DownloadResult struct {
		Value []media.SearchItem
		Error error
	}
)

type Dispatcher struct {
	scrapeCallbacks   []func(item media.SearchItem) ScrapeResult
	scrapeMu          sync.Mutex
	extractCallbacks  []func([]storage.Magnet) ExtractResult
	extractMu         sync.Mutex
	downloadCallbacks []func([]storage.Download) DownloadResult
	downloadMu        sync.Mutex
}

func NewDispatcher() Dispatcher {
	return Dispatcher{}
}

// TODO Add non-fatal hooks
func (d *Dispatcher) OnScrape(f func(item media.SearchItem) ScrapeResult) {
	d.scrapeMu.Lock()
	defer d.scrapeMu.Unlock()
	d.scrapeCallbacks = append(d.scrapeCallbacks, f)
}

func (d *Dispatcher) OnExtract(f func(magnets []storage.Magnet) ExtractResult) {
	d.extractMu.Lock()
	defer d.extractMu.Unlock()
	d.extractCallbacks = append(d.extractCallbacks, f)
}

func (d *Dispatcher) OnDownload(f func(downloads []storage.Download) DownloadResult) {
	d.downloadMu.Lock()
	defer d.downloadMu.Unlock()
	d.downloadCallbacks = append(d.downloadCallbacks, f)
}

// TODO Parallelize work in scraping, extracting and downloading
func (d *Dispatcher) Scrape(item media.SearchItem) ScrapeResult {
	var magnets []storage.Magnet
	for _, f := range d.scrapeCallbacks {
		result := f(item)
		if result.Error != nil {
			return result
		}

		magnets = append(magnets, result.Value...)
	}

	return ScrapeResult{magnets, nil}
}

func (d *Dispatcher) Extract(magnets []storage.Magnet) ExtractResult {
	var downloads []storage.Download
	for _, f := range d.extractCallbacks {
		result := f(magnets)
		if result.Error != nil {
			return result
		}

		downloads = append(downloads, result.Value...)
	}

	return ExtractResult{downloads, nil}
}

func (d *Dispatcher) Download(downloads []storage.Download) DownloadResult {
	var items []media.SearchItem
	for _, f := range d.downloadCallbacks {
		result := f(downloads)
		if result.Error != nil {
			return result
		}

		items = append(items, result.Value...)
	}

	return DownloadResult{items, nil}
}
