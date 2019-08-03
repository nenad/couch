package state

import (
	"github.com/dyrkin/fsm"
	"github.com/nenad/couch/pkg/media"
	"github.com/nenad/couch/pkg/storage"
	"github.com/sirupsen/logrus"
)

const (
	PendingState          = fsm.State("Pending")
	ScrapingState         = fsm.State("Scraping")
	ScrapingErrorState    = fsm.State("ScrapingError")
	ExtractingState       = fsm.State("Extracting")
	ExtractingErrorState  = fsm.State("ExtractingError")
	DownloadingState      = fsm.State("Downloading")
	DownloadingErrorState = fsm.State("DownloadingError")
	DownloadedState       = fsm.State("Downloaded")
)

type Flow struct {
	item         media.SearchItem
	scrapeFunc   func(item media.SearchItem) ScrapeResult
	extractFunc  func([]storage.Magnet) ExtractResult
	downloadFunc func([]storage.Download) DownloadResult

	fsm          *fsm.FSM
	scrapeDone   chan ScrapeResult
	extractDone  chan ExtractResult
	downloadDone chan DownloadResult
}

func (f *Flow) SetScrapeFunc(scraper func(item media.SearchItem) ScrapeResult) {
	f.scrapeFunc = scraper
}

func (f *Flow) SetExtractFunc(extractor func(magnets []storage.Magnet) ExtractResult) {
	f.extractFunc = extractor
}

func (f *Flow) SetDownloadFunc(downloader func(downloads []storage.Download) DownloadResult) {
	f.downloadFunc = downloader
}

func (f *Flow) Begin() {
	// Two updates, since we need to quickly switch two states: None->Pending->Scraping
	f.update()
	f.update()
}

func (f *Flow) Status() fsm.State {
	return f.fsm.CurrentState()
}

func (f *Flow) Resume(state fsm.State, data interface{}) {
	f.fsm.Goto(state).With(data)
	f.update()
}

func New(item media.SearchItem) *Flow {
	f := fsm.NewFSM()

	flow := &Flow{
		item: item,

		fsm:          f,
		scrapeDone:   make(chan ScrapeResult, 1),
		extractDone:  make(chan ExtractResult, 1),
		downloadDone: make(chan DownloadResult, 1),
		scrapeFunc: func(item media.SearchItem) ScrapeResult {
			return ScrapeResult{}
		},
		extractFunc: func(magnets []storage.Magnet) ExtractResult {
			return ExtractResult{}
		},
		downloadFunc: func(downloads []storage.Download) DownloadResult {
			return DownloadResult{}
		},
	}

	f.OnTransition(func(from fsm.State, to fsm.State) {
		logrus.Debugf("%s: Moving from %q to %q", item.Term, from, to)
	})

	f.SetDefaultHandler(func(event *fsm.Event) *fsm.NextState {
		logrus.Errorf("Could not handle event: %#v", event)
		return f.Stay()
	})

	f.StartWith(PendingState, item)

	f.When(PendingState)(func(event *fsm.Event) *fsm.NextState {
		return f.Goto(ScrapingState).With(event.Data)
	})

	f.When(ScrapingState)(func(event *fsm.Event) *fsm.NextState {
		item := event.Data.(media.SearchItem)

		result := flow.scrapeFunc(item)
		flow.scrapeDone <- result

		if result.Error != nil {
			return f.Goto(ScrapingErrorState).With(result.Error)
		}

		return f.Goto(ExtractingState).With(result.Value)
	})

	f.When(ExtractingState)(func(event *fsm.Event) *fsm.NextState {
		magnets := event.Data.([]storage.Magnet)

		result := flow.extractFunc(magnets)
		flow.extractDone <- result

		if result.Error != nil {
			return f.Goto(ExtractingErrorState).With(result.Error)
		}

		return f.Goto(DownloadingState).With(result.Value)
	})

	f.When(DownloadingState)(func(event *fsm.Event) *fsm.NextState {
		item := event.Data.([]storage.Download)

		result := flow.downloadFunc(item)
		flow.downloadDone <- result

		if result.Error != nil {
			return f.Goto(DownloadingErrorState).With(result.Error)
		}

		return f.Goto(DownloadedState)
	})

	f.When(DownloadedState)(func(event *fsm.Event) *fsm.NextState {
		i := event.Message.(media.SearchItem)
		close(flow.scrapeDone)
		close(flow.extractDone)
		close(flow.downloadDone)
		logrus.Infof("Downloaded %q successfully", i.Term)
		return f.Stay()
	})

	f.When(ScrapingErrorState)(func(event *fsm.Event) *fsm.NextState {
		logrus.Errorf("Scraping failed: %#v", event.Data)
		return f.Stay()
	})
	f.When(ExtractingErrorState)(func(event *fsm.Event) *fsm.NextState {
		logrus.Errorf("Extracting failed: %#v", event.Data)
		return f.Stay()
	})
	f.When(DownloadingErrorState)(func(event *fsm.Event) *fsm.NextState {
		logrus.Errorf("Downloading failed: %#v", event.Data)
		return f.Stay()
	})

	// Set up listeners for state changes
	go func() {
	outer:
		for {
			select {
			case r := <-flow.scrapeDone:
				if r.Error != nil {
					flow.fsm.Goto(ScrapingErrorState) // TODO Retry after timeout
				}
				flow.update()
			case r := <-flow.extractDone:
				if r.Error != nil {
					flow.fsm.Goto(ExtractingErrorState) // TODO Retry after timeout
				}
				flow.update()
			case r := <-flow.downloadDone:
				if r.Error != nil {
					flow.fsm.Goto(DownloadingErrorState) // TODO Retry after timeout
				}
				flow.update()
				break outer
			}
		}
	}()

	return flow
}

func (f *Flow) update() {
	f.fsm.Send(f.item)
}
