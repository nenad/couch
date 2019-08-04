package state_test

import (
	"testing"
	"time"

	"github.com/dyrkin/fsm"
	"github.com/nenad/couch/pkg/media"
	"github.com/nenad/couch/pkg/state"
	"github.com/nenad/couch/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestFlow_DefaultStateIsPending(t *testing.T) {
	item := media.NewMovie("Batman", 2010, "tBadman")
	f := state.New(item)
	assert.Equal(t, state.PendingState, f.Status())
}

func TestFlow_BeginFinishesWithDownloadedAfterDelay(t *testing.T) {
	item := media.NewMovie("Batman", 2010, "tBadman")
	f := state.New(item)
	f.Begin()
	time.Sleep(time.Millisecond * 50)
	assert.Equal(t, state.DownloadedState, f.Status())
}

func TestFlow_SetScrapeFunc(t *testing.T) {
	item := media.NewMovie("Batman", 2010, "tBadman")
	f := state.New(item)

	executed := false

	f.SetScrapeFunc(func(item media.SearchItem) state.ScrapeResult {
		executed = true
		return state.ScrapeResult{}
	})
	f.Begin()
	time.Sleep(time.Millisecond * 50)
	assert.True(t, executed)
}

func TestFlow_SetExtractFunc(t *testing.T) {
	item := media.NewMovie("Batman", 2010, "tBadman")
	f := state.New(item)

	executed := false

	f.SetExtractFunc(func(magnets []storage.Magnet) state.ExtractResult {
		executed = true
		return state.ExtractResult{}
	})
	f.Begin()
	time.Sleep(time.Millisecond * 50)
	assert.True(t, executed)
}

func TestFlow_SetDownloadFunc(t *testing.T) {
	item := media.NewMovie("Batman", 2010, "tBadman")
	f := state.New(item)

	executed := false

	f.SetDownloadFunc(func(downloads []storage.Download) state.DownloadResult {
		executed = true
		return state.DownloadResult{}
	})
	f.Begin()
	time.Sleep(time.Millisecond * 50)
	assert.True(t, executed)
}

func TestFlow_ResumeScenarios(t *testing.T) {
	item := media.NewSeason("Gotham", 1, "tGotham")
	magnets := []storage.Magnet{{Item: item}}
	downloads := []storage.Download{{Item: item}}

	testCases := []struct {
		startingState  fsm.State
		data           interface{}
		shouldScrape   bool
		shouldExtract  bool
		shouldDownload bool
	}{
		{
			startingState:  state.ScrapingState,
			data:           item,
			shouldScrape:   true,
			shouldExtract:  true,
			shouldDownload: true,
		},
		{
			startingState:  state.ExtractingState,
			data:           magnets,
			shouldScrape:   false,
			shouldExtract:  true,
			shouldDownload: true,
		},
		{
			startingState:  state.DownloadingState,
			data:           downloads,
			shouldScrape:   false,
			shouldExtract:  false,
			shouldDownload: true,
		},
	}

	for _, c := range testCases {
		f := state.New(item)

		scrapped := false
		f.SetScrapeFunc(func(item media.SearchItem) state.ScrapeResult {
			scrapped = true
			return state.ScrapeResult{}
		})
		extracted := false
		f.SetExtractFunc(func(magnets []storage.Magnet) state.ExtractResult {
			extracted = true
			return state.ExtractResult{}
		})
		downloaded := false
		f.SetDownloadFunc(func(downloads []storage.Download) state.DownloadResult {
			downloaded = true
			return state.DownloadResult{}
		})

		f.Resume(c.startingState, c.data)
		time.Sleep(time.Millisecond * 50)
		assert.Equal(t, c.shouldScrape, scrapped)
		assert.Equal(t, c.shouldExtract, extracted)
		assert.Equal(t, c.shouldDownload, downloaded)
		assert.Equal(t, state.DownloadedState, f.Status())
	}
}
