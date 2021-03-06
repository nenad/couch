package storage

import (
	"database/sql"
	"time"

	"github.com/nenad/couch/pkg/media"
)

const (
	// Possible statuses for the media item
	StatusPending     Status = "Pending"
	StatusScraped     Status = "Scraped"
	StatusExtracting  Status = "Extracting"
	StatusDownloading Status = "Downloading"
	StatusDownloaded  Status = "Downloaded"
	StatusError       Status = "Error"

	Quality4K  Quality = "4K"
	QualityFHD Quality = "FHD"
	QualityHD  Quality = "HD"
	QualitySD  Quality = "SD"

	EncodingXVID Encoding = "XviD"
	Encodingx264 Encoding = "x264"
	Encodingx265 Encoding = "x265"
	EncodingVC1  Encoding = "VC-1"
)

type (
	// Status is the current status of the item
	Status string

	// Media is the model used for storing the item's information
	Media struct {
		Item media.SearchItem

		CreatedAt time.Time
		UpdatedAt time.Time
		Status    Status
	}

	// Quality is the quality of the media
	Quality string

	Encoding string

	Magnet struct {
		Location string
		Quality  Quality
		Encoding Encoding
		Item     media.SearchItem
		Size     uint64 // Size in bytes
		Rating   int
		Seeders  int
	}

	MediaRepository struct {
		db *sql.DB
	}

	// A Download stores the remote and local locations of a file
	Download struct {
		// Remote is the location where the original file resides (ex. URL)
		Remote      string
		// Local is the location where the file will be downloaded
		Local string
		// Item is the metadata about the downloaded file
		Item        media.SearchItem
	}
)

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db}
}

func (r *MediaRepository) StoreItem(item media.SearchItem) error {
	now := time.Now().UTC().Format(ISO8601)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"INSERT INTO search_items (title, type, imdb, created_at, updated_at, status) VALUES (?, ?, ?, ?, ?, ?)",
		item.Term, item.Type, item.IMDb, now, now, StatusPending,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *MediaRepository) AddDownload(download Download) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"INSERT OR IGNORE INTO downloads (title, url, destination, status) VALUES (?, ?, ?, ?)",
		download.Item.Term,
		download.Remote,
		download.Local,
		"Downloading",
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *MediaRepository) AddTorrent(t Magnet) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO torrents (title, url, quality, encoding, rating, size) VALUES (?, ?, ?, ?, ?, ?)`,
		t.Item.Term, t.Location, t.Quality, t.Encoding, t.Rating, t.Size)

	return err
}

func (r *MediaRepository) Delete(title string) error {
	_, err := r.db.Exec("DELETE FROM search_items WHERE title = ?", title)
	return err
}

func (r *MediaRepository) Fetch(title string) (m Media, err error) {
	row := r.db.QueryRow("SELECT title, type, status, imdb, created_at, updated_at FROM search_items WHERE title = ?", title)
	err = row.Scan(&m.Item.Term, &m.Item.Type, &m.Status, &m.Item.IMDb, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *MediaRepository) Status(title string, status Status) error {
	_, err := r.db.Exec("UPDATE search_items SET status = ? WHERE title = ?", status, title)
	return err
}

func (r *MediaRepository) InProgressDownloads() (downloads []Download, err error) {
	query := `SELECT m.title, m.type, l.url, l.destination FROM search_items m
JOIN downloads l on l.title = m.title
WHERE m.status in ('Extracting', 'Downloading', 'Error')
AND l.status in ('Error', 'Downloading');
`

	rows, err := r.db.Query(query)
	if err != nil {
		return
	}

	for rows.Next() {
		var d Download
		err = rows.Scan(&d.Item.Term, &d.Item.Type, &d.Remote, &d.Local)
		if err != nil {
			return
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}

func (r *MediaRepository) NonExtractedTorrents() (torrents []Magnet, err error) {
	query := `SELECT m.title, m.type, t.url, t.size, t.quality, t.encoding, t.rating FROM search_items m
JOIN torrents t on t.title = m.title
WHERE m.status in ('Extracting', 'Scraped', 'Pending')
AND m.title NOT IN (SELECT l.title FROM downloads l)
GROUP BY t.title
ORDER BY t.rating ASC;
`

	rows, err := r.db.Query(query)
	if err != nil {
		return
	}

	for rows.Next() {
		var t Magnet
		err = rows.Scan(&t.Item.Term, &t.Item.Type, &t.Location, &t.Size, &t.Quality, &t.Encoding, &t.Rating)
		if err != nil {
			return
		}
		torrents = append(torrents, t)
	}
	return torrents, nil
}

func (r *MediaRepository) UpdateDownload(term, url string, isDone bool, err error) error {
	status := "Downloading"
	if err != nil {
		status = "Error"
	} else if isDone {
		status = "Downloaded"
	}

	tx, err := r.db.Begin()

	_, err = tx.Exec(
		"UPDATE downloads SET status = ? WHERE url = ?",
		status,
		url,
	)

	if err != nil {
		tx.Rollback()
		return err
	}

	var errors, downloading, downloaded int

	row := tx.QueryRow(`SELECT 
       count(CASE WHEN status = 'Error' THEN status END) as error,
       count(CASE WHEN status = 'Downloading' THEN status END) as downloading,
       count(CASE WHEN status = 'Downloaded' THEN status END) as downloaded
FROM downloads WHERE title = ?`, term)

	if err := row.Scan(&errors, &downloading, &downloaded); err != nil {
		tx.Rollback()
		return err
	}

	if errors > 0 {
		status = "Error"
	} else if downloading > 0 {
		status = "Downloading"
	}

	if _, err := tx.Exec("UPDATE search_items SET status = ? WHERE title = ?", status, term); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *MediaRepository) GetAvailableMagnet(title string) (m string, err error) {
	row := r.db.QueryRow(`SELECT t.url FROM torrents t WHERE t.title = ? ORDER BY t.rating ASC LIMIT 1;`, title)
	err = row.Scan(&m)
	return m, err
}

func (r *MediaRepository) ItemByLocation(path string) (m media.SearchItem, err error) {
	row := r.db.QueryRow(`SELECT s.title, s.type, s.imdb 
FROM search_items s
JOIN downloads r on s.title = r.title
WHERE r.url = ?`, path)
	err = row.Scan(&m.Term, &m.Type, &m.IMDb)
	return m, err
}
