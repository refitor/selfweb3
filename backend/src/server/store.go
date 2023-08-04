package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/joyrexus/buckets"
)

type db_bolt struct {
	db         *buckets.DB
	bucketsMap sync.Map
}

func boltDBInit(path string) (*db_bolt, error) {
	p := &db_bolt{}
	db, err := buckets.Open(path)
	if err != nil {
		return nil, err
	}
	p.db = db
	return p, nil
}

func (p *db_bolt) setBucket(name string, bucket *buckets.Bucket) {
	p.bucketsMap.Store(name, bucket)
}

func (p *db_bolt) getBucket(name string) *buckets.Bucket {
	if d, ok := p.bucketsMap.Load(name); ok {
		return d.(*buckets.Bucket)
	}
	return nil
}

func (p *db_bolt) DBCreate(name string) error {
	if p.db == nil {
		return fmt.Errorf("invalid db, name: %v", name)
	}

	bucket, err := p.db.New([]byte(name))
	if err == nil {
		p.setBucket(name, bucket)
	}
	return err
}

func (p *db_bolt) DBClose() error {
	if p.db == nil {
		return errors.New("invalid db")
	}
	return p.db.Close()
}

func (p *db_bolt) DBPut(name, k string, v interface{}) error {
	bucket := p.getBucket(name)
	if bucket == nil {
		return fmt.Errorf("invalid bucket, name: %s", name)
	}

	if _, ok := v.([]byte); ok {
		return bucket.Put([]byte(k), v.([]byte))
	} else {
		vbuf, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(k), vbuf)
	}
}

func (p *db_bolt) DBRange(name string, f func(k string, v []byte) bool) error {
	bucket := p.getBucket(name)
	if bucket == nil {
		return fmt.Errorf("invalid bucket, name: %s", name)
	}

	items, err := bucket.Items()
	if err != nil {
		return err
	}
	for _, item := range items {
		if f != nil && !f(string(item.Key), item.Value) {
			break
		}
	}
	return nil
}

func (p *db_bolt) DBRangeByPrefix(name, prefix string, f func(k string, v []byte) bool) error {
	bucket := p.getBucket(name)
	if bucket == nil {
		return fmt.Errorf("invalid bucket, name: %s", name)
	}

	items, err := bucket.NewPrefixScanner([]byte(prefix)).Items()
	if err != nil {
		return err
	}

	for _, item := range items {
		if f != nil && !f(string(item.Key), item.Value) {
			break
		}
	}
	return nil
}

func (p *db_bolt) DBGet(name, k string) (ret []byte, err error) {
	bucket := p.getBucket(name)
	if bucket == nil {
		return nil, fmt.Errorf("invalid bucket, name: %s", name)
	}
	ret, err = bucket.Get([]byte(k))
	return
}

func (p *db_bolt) DBDel(name, k string) error {
	bucket := p.getBucket(name)
	if bucket == nil {
		return fmt.Errorf("invalid bucket, name: %s", name)
	}
	return bucket.Delete([]byte(k))
}
