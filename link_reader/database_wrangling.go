package linkreader

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DatabaseConnector struct {
	db *sql.DB
}

type LinkResolution int

const (
	LinkUnprocessed LinkResolution = iota
	LinkDismissed
	LinkSaved
	LinkSnoozed
)

type DatabaseStats struct {
	Unprocessed int
	Dismissed   int
	Saved       int
	Snoozed     int
}

func OpenConnection(filename string) (*DatabaseConnector, error) {
	conn := DatabaseConnector{}
	var err error

	conn.db, err = sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	create_tables := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS links (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT,
		link TEXT,
		resolution INTEGER DEFAULT %d,
		source_id INTEGER,
		FOREIGN KEY(source_id) REFERENCES sources(id)
	);

	CREATE TABLE IF NOT EXISTS tags_list (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT UNIQUE
	);
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER NOT NULL PRIMARY KEY,
		link_id INTEGER,
		tag_id INTEGER,
		FOREIGN KEY(link_id) REFERENCES links(id),
		FOREIGN KEY(tag_id) REFERENCES tags_list(id),
		UNIQUE (link_id, tag_id)
	);
	
	CREATE TABLE IF NOT EXISTS filenames (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT UNIQUE,
		length INTEGER,
		current_position
		);
	
	CREATE TABLE IF NOT EXISTS sources (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT UNIQUE
	);
	`, LinkUnprocessed)

	_, err = conn.db.Exec(create_tables)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func (conn *DatabaseConnector) Close() error {
	err := conn.db.Close()
	return err
}

func (conn *DatabaseConnector) GetStats() (*DatabaseStats, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	query, err := tx.Prepare(`SELECT COALESCE(SUM(CASE WHEN resolution = ? THEN 1 ELSE 0 END), 0) as unprocessed,
									 COALESCE(SUM(CASE WHEN resolution = ? THEN 1 ELSE 0 END), 0) as dismissed,
									 COALESCE(SUM(CASE WHEN resolution = ? THEN 1 ELSE 0 END), 0) as saved,
									 COALESCE(SUM(CASE WHEN resolution = ? THEN 1 ELSE 0 END), 0) as snoozed
		FROM links`)
	if err != nil {
		return nil, err
	}
	defer query.Close()

	row := query.QueryRow(LinkUnprocessed, LinkDismissed, LinkSaved, LinkSnoozed)

	var stats DatabaseStats
	err = row.Scan(&(stats.Unprocessed), &(stats.Dismissed), &(stats.Saved), &(stats.Snoozed))
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (conn *DatabaseConnector) InsertLink(link Linker) (int, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Commit()

	insert_links_query, err := tx.Prepare("INSERT INTO links (name, link) VALUES (?, ?) RETURNING id;")
	if err != nil {
		return -1, err
	}
	defer insert_links_query.Close()

	response, err := insert_links_query.Exec(link.GetName(), link.GetHREF())
	if err != nil {
		return -1, err
	}
	returned_id, err := response.LastInsertId()
	if err != nil {
		return -1, err
	}

	return int(returned_id), nil
}

func (conn *DatabaseConnector) GetUnresolvedLink() (int, Linker, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return -1, nil, err
	}
	defer tx.Commit()

	query, err := tx.Prepare("SELECT id, name, link FROM links WHERE resolution = ? LIMIT 1;")
	if err != nil {
		return -1, nil, err
	}
	defer query.Close()

	row := query.QueryRow(LinkUnprocessed)

	var id int
	var link Link
	err = row.Scan(&id, &(link.name), &(link.href))
	if err != nil {
		return -1, nil, err
	}

	return id, link, nil
}

func (conn *DatabaseConnector) GetAllSavedLinks() ([]int, []Linker, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Commit()

	query, err := tx.Prepare("SELECT id, name, link FROM links WHERE resolution = ?;")
	if err != nil {
		return nil, nil, err
	}
	defer query.Close()

	rows, err := query.Query(LinkSaved)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	results := make([]Linker, 0)

	for rows.Next() {
		var id int
		result := Link{}

		err = rows.Scan(&id, &(result.name), &(result.href))
		if err != nil {
			return nil, nil, err
		}

		ids = append(ids, id)
		results = append(results, result)
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	return ids, results, nil
}

func (conn *DatabaseConnector) ResetSnoozedLinks() error {
	tx, err := conn.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	query, err := tx.Prepare("UPDATE links SET resolution = ? WHERE resolution = ?;")
	if err != nil {
		return err
	}
	defer query.Close()

	_, err = query.Exec(LinkUnprocessed, LinkSnoozed)
	if err != nil {
		return err
	}

	return nil
}

func (conn *DatabaseConnector) GetSimilarLinks(link Linker) ([]int, []Linker, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Commit()

	query, err := tx.Prepare("SELECT id, name, link FROM links WHERE link LIKE CONCAT(?, '%') AND resolution NOT IN (?, ?);")
	if err != nil {
		return nil, nil, err
	}
	defer query.Close()

	rows, err := query.Query(string(link.GetHREF()), LinkUnprocessed, LinkDismissed)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	results := make([]Linker, 0)

	for rows.Next() {
		var id int
		var name []byte
		var link []byte

		err = rows.Scan(&id, &name, &link)
		if err != nil {
			return nil, nil, err
		}

		ids = append(ids, id)
		results = append(results, Link{name, link})
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	return ids, results, nil
}

func (conn *DatabaseConnector) GetSimilarLinksContext(ids []int) ([]Linker, error) {
	query := "SELECT (id, name, link) FROM links WHERE id IN ("
	for id := range ids {
		query += fmt.Sprintf("%v, ", id)
	}
	query += ");"

	tx, err := conn.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	rows, err := conn.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]Linker, 0)

	for rows.Next() {
		var id int
		var name []byte
		var link []byte

		err = rows.Scan(&id, &name, &link)
		if err != nil {
			return nil, err
		}

		results[id] = Link{name, link}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (conn *DatabaseConnector) MarkLinkById(id int, resolution LinkResolution) error {
	tx, err := conn.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	query, err := tx.Prepare("UPDATE links SET resolution = ? WHERE id = ?;")
	if err != nil {
		return err
	}
	defer query.Close()

	_, err = query.Exec(resolution, id)
	if err != nil {
		return err
	}

	return nil
}

// used for undo - looks for the last marked link, then unmarks it.
// also returns the id of the unmarked link. if none links are marked, returns err
func (conn *DatabaseConnector) UnmarkLastLink() (int, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Commit()

	query, err := tx.Prepare("UPDATE links SET resolution = ? WHERE id = (SELECT id FROM links WHERE resolution != ? ORDER BY id DESC LIMIT 1) RETURNING id;")
	if err != nil {
		return -1, err
	}
	defer query.Close()

	result := query.QueryRow(LinkUnprocessed, LinkUnprocessed)
	var id int
	err = result.Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (conn *DatabaseConnector) TagLink(link_id int, tag_name string) error {
	// log.Debug("trying to apply a tag", "tag_name", tag_name)
	tx, err := conn.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	query, err := tx.Prepare("INSERT INTO tags_list (name) VALUES (?) ON CONFLICT DO NOTHING;")
	if err != nil {
		return err
	}
	defer query.Close()

	_, err = query.Exec(tag_name)
	if err != nil {
		return err
	}
	query, err = tx.Prepare("SELECT id FROM tags_list ORDER BY id DESC LIMIT 1;")
	row := query.QueryRow()
	var tag_id int
	row.Scan(&tag_id)
	// log.Debug("tag id!", "tag_id", tag_id)

	query, err = tx.Prepare("INSERT INTO tags (link_id, tag_id) VALUES (?, ?);")
	_, err = query.Exec(link_id, tag_id)
	if err != nil {
		return err
	}

	return nil
}

func (conn *DatabaseConnector) UntagLink(link_id int) error {
	// log.Debug("trying to apply a tag", "tag_name", tag_name)
	tx, err := conn.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	query, err := tx.Prepare("DELETE FROM tags WHERE link_id = ?;")
	if err != nil {
		return err
	}
	defer query.Close()

	_, err = query.Exec(link_id)
	if err != nil {
		return err
	}

	return nil
}
func (conn *DatabaseConnector) GetLinkTags(linkID int) ([]string, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	query, err := tx.Prepare("SELECT tags_list.name FROM tags LEFT JOIN tags_list on tags.tag_id = tags_list.id WHERE tags.link_id = ?")
	if err != nil {
		return nil, err
	}
	defer query.Close()

	rows, err := query.Query(linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]string, 0)

	for rows.Next() {
		var tag string

		err = rows.Scan(&tag)
		if err != nil {
			return nil, err
		}

		results = append(results, tag)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, nil
}
func (conn *DatabaseConnector) MarkLinkSource(link_id int, source string) error {
	// log.Debug("trying to apply a tag", "tag_name", tag_name)
	tx, err := conn.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	query, err := tx.Prepare("INSERT INTO sources (name) VALUES (?) ON CONFLICT DO NOTHING;")
	if err != nil {
		return err
	}
	defer query.Close()

	_, err = query.Exec(source)
	if err != nil {
		return err
	}
	query, err = tx.Prepare("SELECT id FROM sources ORDER BY id DESC LIMIT 1;")
	row := query.QueryRow()
	var sourceID int
	row.Scan(&sourceID)

	query, err = tx.Prepare("UPDATE links SET source_id = ? WHERE id = ?;")
	_, err = query.Exec(sourceID, link_id)
	if err != nil {
		return err
	}

	return nil
}
