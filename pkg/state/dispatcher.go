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
	scrapeCallbacks      []func(item media.SearchItem) ScrapeResult
	afterScrapeCallbacks []func([]storage.Magnet) []storage.Magnet
	scrapeMu             sync.Mutex

	extractCallbacks      []func([]storage.Magnet) ExtractResult
	afterExtractCallbacks []func([]storage.Download) []storage.Download
	extractMu             sync.Mutex

	downloadCallbacks      []func([]storage.Download) DownloadResult
	afterDownloadCallbacks []func([]media.SearchItem) []media.SearchItem
	downloadMu             sync.Mutex
}

func NewDispatcher() Dispatcher {
	return Dispatcher{}
}

// OnScrape registers a callback to be invoked when scraping begins
func (d *Dispatcher) OnScrape(f func(item media.SearchItem) ScrapeResult) {
	d.scrapeMu.Lock()
	defer d.scrapeMu.Unlock()
	d.scrapeCallbacks = append(d.scrapeCallbacks, f)
}

// AfterScrape registers a callback to be invoked when scraping finishes
func (d *Dispatcher) AfterScrape(f func([]storage.Magnet) []storage.Magnet) {
	d.scrapeMu.Lock()
	defer d.scrapeMu.Unlock()
	d.afterScrapeCallbacks = append(d.afterScrapeCallbacks, f)
}

// OnExtract registers a callback to be invoked when extracting begins
func (d *Dispatcher) OnExtract(f func(magnets []storage.Magnet) ExtractResult) {
	d.extractMu.Lock()
	defer d.extractMu.Unlock()
	d.extractCallbacks = append(d.extractCallbacks, f)
}

// AfterExtract registers a callback to be invoked when extracting finishes
func (d *Dispatcher) AfterExtract(f func([]storage.Download) []storage.Download) {
	d.extractMu.Lock()
	defer d.extractMu.Unlock()
	d.afterExtractCallbacks = append(d.afterExtractCallbacks, f)
}

// OnDownload registers a callback to be invoked when downloading begins
func (d *Dispatcher) OnDownload(f func(downloads []storage.Download) DownloadResult) {
	d.downloadMu.Lock()
	defer d.downloadMu.Unlock()
	d.downloadCallbacks = append(d.downloadCallbacks, f)
}

// AfterDownload registers a callback to be invoked when downloading finishes
func (d *Dispatcher) AfterDownload(f func([]media.SearchItem) []media.SearchItem) {
	d.downloadMu.Lock()
	defer d.downloadMu.Unlock()
	d.afterDownloadCallbacks = append(d.afterDownloadCallbacks, f)
}

// Scrape invokes all registered hooks through OnScrape in the order
// they were registered. If a callback returns error, this function
// will return the result of the errored callback.
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

// Extract invokes all registered hooks through OnExtract in the order
// they were registered. If a callback returns error, this function
// will return the result of the errored callback.
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

// Download invokes all registered hooks through OnDownload in the order
// they were registered. If a callback returns error, this function
// will return the result of the errored callback.
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
