package kvs

import (
	badger "github.com/dgraph-io/badger/v3"
)

type KVEntry struct {
	Key   string
	Value []byte
}

func (kv *KVEntry) Fetch(db *badger.DB) error {
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(kv.Key))
		if err != nil {
			return err
		}

		innerVal := make([]byte, item.ValueSize())

		item.Value(func(val []byte) error {
			copy(innerVal, val)
			kv.Value = innerVal
			return nil
		})
		return err
	})
	return err
}

func (kv *KVEntry) Store(db *badger.DB) error {
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(kv.Key), kv.Value)
		return err
	})

	return err
}

func NewFromStore(key string, db *badger.DB) (*KVEntry, error) {
	kve := &KVEntry{}
	kve.Key = key

	err := kve.Fetch(db)
	if err != nil {
		return kve, err
	}

	return kve, nil
}

func New(f string, memory bool) (*badger.DB, error) {
	var db *badger.DB
	var err error
	if memory {
		db, err = badger.Open(badger.DefaultOptions("").WithInMemory(true))
	} else {
		db, err = badger.Open(badger.DefaultOptions(f))
	}

	return db, err
}
