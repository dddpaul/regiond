package utils

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

// InitDb opens Bolt database
func InitDb() *bolt.DB {
	db, err := bolt.Open("fedpa.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// CreateBucket creates bucket if it doesn't exist
func CreateBucket(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Upstreams"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
}

// GetFromBucket reads byte slice from bucket by key
func GetFromBucket(db *bolt.DB, key string) []byte {
	var byt []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Upstreams"))
		byt = b.Get([]byte(key))
		return nil
	})
	return byt
}

// PutToBucket writes byte slice to bucket
func PutToBucket(db *bolt.DB, key string, val []byte) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Upstreams"))
		return b.Put([]byte(key), val)
	})
}
