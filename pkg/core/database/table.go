package database

// Table is an abstract data structure built on top of a db
type Table struct {
	prefix []byte
	db     Database
}

// NewTable create a new Table
func NewTable(db Database, prefix []byte) *Table {
	return &Table{
		prefix,
		db,
	}
}

// Has returns true if the db does contains the given key
func (t *Table) Has(key []byte) (bool, error) {
	key = append(t.prefix, key...)
	return t.db.Has(key)
}

// Put sets the value for the given key. It overwrites any previous value.
func (t *Table) Put(key []byte, value []byte) error {
	key = append(t.prefix, key...)
	return t.db.Put(key, value)
}

// Write apply the given batch to the db
func (t *Table) Write(bvs *batchValues) error {
	return t.db.Write(bvs)
}

// Get gets the value for the given key
func (t *Table) Get(key []byte) ([]byte, error) {
	key = append(t.prefix, key...)
	return t.db.Get(key)
}

// Delete deletes the value for the given key
func (t *Table) Delete(key []byte) error {
	key = append(t.prefix, key...)
	return t.db.Delete(key)
}

// Close closes the db
func (t *Table) Close() error {
	return t.db.Close()
}
