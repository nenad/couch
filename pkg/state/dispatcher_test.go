package state_test

import (
	"fmt"
	"testing"

	"github.com/nenad/couch/pkg/media"
	"github.com/nenad/couch/pkg/state"
	"github.com/nenad/couch/pkg/storage"
	"github.com/stretchr/testify/assert"
)

var items = []media.SearchItem{
	media.NewMovie("Batman", 2010, "tBatman"),
	media.NewEpisode("Superman", 1, 3, "tSuperman"),
}

func TestDispatcher_Scrape(t *testing.T) {
	d := state.NewDispatcher()
	expItem := items[0]
	expMagnets := []storage.Magnet{
		{Item: expItem, Location: "magnet://1"},
		{Item: expItem, Location: "magnet://2"},
	}

	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Value: []storage.Magnet{expMagnets[0]}}
	})
	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Value: []storage.Magnet{expMagnets[1]}}
	})

	res := d.Scrape(expItem)

	assert.Len(t, res.Value, len(expMagnets), "difference between returned values from dispatcher")
	for _, m := range expMagnets {
		assert.Contains(t, res.Value, m)
	}
}

func TestDispatcher_ScrapeError(t *testing.T) {
	d := state.NewDispatcher()
	expItem := items[0]
	expMagnets := []storage.Magnet{
		{Item: expItem, Location: "magnet://1"},
		{Item: expItem, Location: "magnet://2"},
	}

	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Value: []storage.Magnet{expMagnets[0]}}
	})
	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Error: fmt.Errorf("something bad happened")}
	})

	res := d.Scrape(expItem)

	assert.Equal(t, state.ScrapeResult{
		Value: nil,
		Error: fmt.Errorf("something bad happened"),
	}, res)
}

func TestDispatcher_Extract(t *testing.T) {
	d := state.NewDispatcher()

	// Episode extractor
	d.OnExtract(func(magnets []storage.Magnet) state.ExtractResult {
		var downloads []storage.Download
		for _, d := range magnets {
			if d.Item.Type == media.TypeEpisode {
				downloads = append(downloads, storage.Download{Item: d.Item})
			}
		}

		return state.ExtractResult{Value: downloads}
	})

	// Movie extractor
	d.OnExtract(func(magnets []storage.Magnet) state.ExtractResult {
		var downloads []storage.Download
		for _, d := range magnets {
			if d.Item.Type == media.TypeMovie {
				downloads = append(downloads, storage.Download{Item: d.Item})
			}
		}

		return state.ExtractResult{Value: downloads}
	})

	expCount := len(items)
	res := d.Extract([]storage.Magnet{
		{Item: items[0]},
		{Item: items[1]},
	})

	var extractedItems []media.SearchItem
	for _, v := range res.Value {
		extractedItems = append(extractedItems, v.Item)
	}

	assert.Len(t, extractedItems, expCount, "difference between returned values from dispatcher")
	for _, i := range items {
		assert.Contains(t, extractedItems, i)
	}
}

func TestDispatcher_ExtractError(t *testing.T) {
	d := state.NewDispatcher()

	// Episode extractor
	d.OnExtract(func(magnets []storage.Magnet) state.ExtractResult {
		return state.ExtractResult{Value: []storage.Download{}}
	})

	// Movie extractor
	d.OnExtract(func(magnets []storage.Magnet) state.ExtractResult {
		return state.ExtractResult{Error: fmt.Errorf("extraction failed")}
	})

	res := d.Extract([]storage.Magnet{
		{Item: items[0]},
		{Item: items[1]},
	})

	assert.Equal(t, state.ExtractResult{Value: nil, Error: fmt.Errorf("extraction failed")}, res)
}

func TestDispatcher_Download(t *testing.T) {
	d := state.NewDispatcher()

	// Episode downloader
	d.OnDownload(func(downloads []storage.Download) state.DownloadResult {
		var downloadedItems []media.SearchItem
		for _, d := range downloads {
			if d.Item.Type == media.TypeEpisode {
				downloadedItems = append(downloadedItems, d.Item)
			}
		}

		return state.DownloadResult{Value: downloadedItems}
	})

	// Movie downloader
	d.OnDownload(func(downloads []storage.Download) state.DownloadResult {
		var downloadedItems []media.SearchItem
		for _, d := range downloads {
			if d.Item.Type == media.TypeMovie {
				downloadedItems = append(downloadedItems, d.Item)
			}
		}

		return state.DownloadResult{Value: downloadedItems}
	})

	expCount := len(items)
	res := d.Download([]storage.Download{
		{Item: items[0]},
		{Item: items[1]},
	})

	assert.Len(t, res.Value, expCount, "difference between returned values from dispatcher")
	for _, i := range items {
		assert.Contains(t, res.Value, i)
	}
}

func TestDispatcher_DownloadError(t *testing.T) {
	d := state.NewDispatcher()

	// Episode downloader
	d.OnDownload(func(downloads []storage.Download) state.DownloadResult {
		return state.DownloadResult{Value: items}
	})

	// Movie downloader
	d.OnDownload(func(downloads []storage.Download) state.DownloadResult {
		return state.DownloadResult{Error: fmt.Errorf("download error")}
	})

	res := d.Download([]storage.Download{
		{Item: items[0]},
		{Item: items[1]},
	})

	assert.Equal(t, state.DownloadResult{Value: nil, Error: fmt.Errorf("download error")}, res)
}

func TestDispatcher_AfterScrape(t *testing.T) {
	d := state.NewDispatcher()
	expItem := items[0]
	expMagnets := []storage.Magnet{
		{Item: expItem, Location: "magnet://1"},
		{Item: expItem, Location: "magnet://2"},
	}

	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Value: []storage.Magnet{expMagnets[0]}}
	})
	d.OnScrape(func(item media.SearchItem) state.ScrapeResult {
		return state.ScrapeResult{Value: []storage.Magnet{expMagnets[1]}}
	})
	d.AfterScrape(func(magnets []storage.Magnet) []storage.Magnet {
		var newMagnets []storage.Magnet
		for _, m := range magnets {
			if m.Location == "magnet://1" {
				newMagnets = append(newMagnets, m)
			}
		}
		return newMagnets
	})

	res := d.Scrape(expItem)

	assert.Len(t, res.Value, 1, "difference between returned values from dispatcher")
	assert.Equal(t, expMagnets[0], res.Value[0])
}

func TestDispatcher_AfterExtract(t *testing.T) {
	d := state.NewDispatcher()

	want := []storage.Download{
		{
			Item:  items[0],
			Local: "/home/media/" + items[0].Term,
		},
		{
			Item:  items[1],
			Local: "/home/media/" + items[1].Term,
		},
	}

	// Episode extractor
	d.OnExtract(func(magnets []storage.Magnet) state.ExtractResult {
		var downloads []storage.Download
		for _, d := range magnets {
			downloads = append(downloads, storage.Download{Item: d.Item, Local: d.Item.Term})
		}
		return state.ExtractResult{Value: downloads}
	})
	d.AfterExtract(func(downloads []storage.Download) []storage.Download {
		for i, d := range downloads {
			downloads[i].Local = "/home/media/" + d.Local
		}
		return downloads
	})

	res := d.Extract([]storage.Magnet{
		{Item: items[0]},
		{Item: items[1]},
	})

	assert.Equal(t, want, res.Value)
	assert.Nil(t, res.Error)
}

func TestDispatcher_AfterDownload(t *testing.T) {
	d := state.NewDispatcher()

	want := []media.SearchItem{
		items[0],
	}

	d.OnDownload(func(downloads []storage.Download) state.DownloadResult {
		var downloadedItems []media.SearchItem
		for _, d := range downloads {
			downloadedItems = append(downloadedItems, d.Item)
		}

		return state.DownloadResult{Value: downloadedItems}
	})

	// Return only the first item
	d.AfterDownload(func(items []media.SearchItem) []media.SearchItem {
		var newItems []media.SearchItem
		for _, i := range items {
			if i.Term == items[0].Term {
				newItems = append(newItems, items[0])
			}
		}
		return newItems
	})

	res := d.Download([]storage.Download{
		{Item: items[0]},
		{Item: items[1]},
	})

	assert.Equal(t, want, res.Value)
	assert.Nil(t, res.Error)
}
