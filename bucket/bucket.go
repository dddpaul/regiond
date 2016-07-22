package bucket

import (
	"fmt"

	"github.com/boltdb/bolt"
)

// CreateBucket creates bucket if it doesn't exist
func Create(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Upstreams"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
}

// GetFromBucket reads byte slice from bucket by key
func Get(db *bolt.DB, key string) []byte {
	var byt []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Upstreams"))
		byt = b.Get([]byte(key))
		return nil
	})
	return byt
}

// PutToBucket writes byte slice to bucket
func Put(db *bolt.DB, key string, val []byte) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Upstreams"))
		return b.Put([]byte(key), val)
	})
}
