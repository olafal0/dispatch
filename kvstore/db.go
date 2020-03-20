package kvstore

import (
	"bytes"
	"database/sql"
	"encoding/gob"

	// Import sqlite3 database driver
	_ "github.com/mattn/go-sqlite3"
)

// KeyValueDB is an object similar to sql.DB that provides simple methods for create,
// read, update, and delete functionality on key-value items.
type KeyValueDB struct {
	db *sql.DB
}

// NewDB initializes a new KeyValueDB database connection.
func NewDB(name string) (kv *KeyValueDB, err error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv(
		table_name TEXT NOT NULL,
		id TEXT NOT NULL,
		val BLOB,
		PRIMARY KEY (table_name, id)
	);`)
	if err != nil {
		return nil, err
	}

	kv = &KeyValueDB{db}
	return kv, nil
}

// SetObject creates or updates the key-value pair.
func (kv *KeyValueDB) SetObject(table, id string, value interface{}) error {
	gobBuffer := new(bytes.Buffer)
	gobEncoder := gob.NewEncoder(gobBuffer)
	err := gobEncoder.Encode(value)
	if err != nil {
		return err
	}

	_, err = kv.db.Exec(
		"INSERT OR REPLACE INTO kv (table_name, id, val) VALUES(?, ?, ?);",
		table, id, gobBuffer.Bytes(),
	)
	if err != nil {
		return err
	}

	return nil
}

// GetObject retrieves and decodes the stored value into result.
func (kv *KeyValueDB) GetObject(table, id string, result interface{}) (err error) {
	row := kv.db.QueryRow(
		"SELECT val FROM kv WHERE table_name = ? AND id = ?",
		table, id,
	)
	var buf []byte
	err = row.Scan(&buf)
	if err != nil {
		return err
	}

	gobBuffer := bytes.NewBuffer(buf)
	gobDecoder := gob.NewDecoder(gobBuffer)
	err = gobDecoder.Decode(result)
	return err
}

// DeleteObject removes an object from the database.
func (kv *KeyValueDB) DeleteObject(table, id string) (err error) {
	_, err = kv.db.Exec(
		"DELETE FROM kv WHERE table_name = ? AND id = ?",
		table, id,
	)
	return err
}

// Table returns the KeyValueTable associated with this table name.
func (kv *KeyValueDB) Table(table string) *KeyValueTable {
	return &KeyValueTable{
		db:    kv,
		Table: table,
	}
}

// KeyValueTable represents a specific table name of the database.
type KeyValueTable struct {
	Table string
	db    *KeyValueDB
}

// GetObject retrieves and decodes the stored value into result.
func (kvt *KeyValueTable) GetObject(id string, result interface{}) error {
	return kvt.db.GetObject(kvt.Table, id, result)
}

// SetObject creates or updates the key-value pair in this table.
func (kvt *KeyValueTable) SetObject(id string, value interface{}) error {
	return kvt.db.SetObject(kvt.Table, id, value)
}

// DeleteObject removes an object from the database.
func (kvt *KeyValueTable) DeleteObject(id string) error {
	return kvt.db.DeleteObject(kvt.Table, id)
}

// IsErrNoRows returns true if the passed error is an sql.ErrNoRows error.
func IsErrNoRows(err error) bool {
	return err == sql.ErrNoRows
}
