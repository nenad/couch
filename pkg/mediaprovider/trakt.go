package mediaprovider

import (
	"github.com/nenadstojanovikj/couch/pkg/media"
	"github.com/nenadstojanovikj/trakt"
	"github.com/sirupsen/logrus"
	"time"
)

func NewTraktProvider(trakt *trakt.Client) *TraktProvider {
	return &TraktProvider{trakt: trakt}
}

type TraktProvider struct {
	trakt *trakt.Client
}

func (p *TraktProvider) Poll() (metadata []media.SearchItem, err error) {
	var removeMeta trakt.FullMetadata

	yesterday := time.Now().Add(-time.Hour * 24).Format("2006-01-02")
	episodes, err := p.trakt.Calendar(yesterday, 1)
	if err != nil {
		return nil, err
	}
	for _, e := range episodes {
		metadata = append(metadata, media.NewEpisode(e.Show.Title, e.Episode.Season, e.Episode.Number, e.Show.IDs.IMDb))
		removeMeta.Episodes = append(removeMeta.Episodes, e.Episode)
	}

	watchEpisodes, err := p.trakt.WatchlistEpisodes()
	if err != nil {
		return nil, err
	}
	for _, e := range watchEpisodes {
		metadata = append(metadata, media.NewEpisode(e.Show.Title, e.Episode.Season, e.Episode.Number, e.Show.IDs.IMDb))
		removeMeta.Episodes = append(removeMeta.Episodes, e.Episode)
	}

	movies, err := p.trakt.WatchlistMovies()
	if err != nil {
		return nil, err
	}
	for _, m := range movies {
		metadata = append(metadata, media.NewMovie(m.Movie.Title, m.Movie.Year, m.Movie.IDs.IMDb))
		removeMeta.Movies = append(removeMeta.Movies, m.Movie)
	}

	seasons, err := p.trakt.WatchlistSeasons()
	if err != nil {
		return nil, err
	}
	for _, s := range seasons {
		metadata = append(metadata, media.NewSeason(s.Show.Title, s.Season.Number, s.Show.IDs.IMDb))
		removeMeta.Seasons = append(removeMeta.Seasons, s.Season)
	}

	if err := p.trakt.RemoveFromWatchlist(removeMeta); err != nil {
		logrus.Errorf("could not remove metadata about items: %s", err)
		return nil, err
	}

	return metadata, nil
}

func (p *TraktProvider) Interval() time.Duration {
	return time.Hour * 1
}