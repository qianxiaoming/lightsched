package data

import (
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// BoltDB 是对go.etcd.io/bbolt的简单封装
type BoltDB struct {
	*bolt.DB
}

var errKeyNotExists error = errors.New("key not found")

func (db *BoltDB) createBucket(name string) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(name))
		if err != nil {
			return fmt.Errorf("Failed to create bucket in database: %s", err)
		}
		return nil
	})
}

func (db *BoltDB) put(bucket string, key string, value []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		err := b.Put([]byte(key), value)
		return err
	})
}

func (db *BoltDB) putJSON(bucket string, key string, value interface{}) ([]byte, error) {
	var err error
	if b, err := json.Marshal(value); err == nil {
		return b, db.put(bucket, key, b)
	}
	return nil, err
}

func (db *BoltDB) getJSON(bucket string, key string, value interface{}) (bool, error) {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v := b.Get([]byte(key))
		if v == nil {
			return errKeyNotExists
		}
		if err := json.Unmarshal([]byte(v), value); err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	} else if err == errKeyNotExists {
		return false, nil
	} else {
		return true, err
	}
}

func (db *BoltDB) getBucketJSON(bucket string, create func() interface{}, save func(interface{})) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var obj interface{} = create()
			if err := json.Unmarshal([]byte(v), obj); err == nil {
				save(obj)
			} else {
				return fmt.Errorf("failed to unmarshal from JSON: %v", err)
			}
		}
		return nil
	})
}
