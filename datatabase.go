package main

import (
	"errors"
	"log"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var (
	db             *bbolt.DB
	dbFile         string
	errInvalidPath = errors.New("invalid path")
	errKeyExists   = errors.New("key exists")
	nodeMap        = make(map[string]TreeNode)
)

func openDB(file string) error {
	var err error
	if db != nil {
		closeDB()
	}
	db, err = bbolt.Open(file, 0o666, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	dbFile = file
	return nil
}

func closeDB() {
	if db != nil {
		db.Close() //nolint:errcheck
	}
}

func mapNodes(nodes []*TreeNode) {
	for _, node := range nodes {
		path := []string{}
		for _, part := range node.Path {
			path = append(path, string(part))
		}
		nodeMap[strings.Join(path, "/")] = *node
		mapNodes(node.Children)
	}
}

func getNodes() []*TreeNode {
	allNodes := []*TreeNode{}
	if db == nil {
		return allNodes
	}
	db.View(func(tx *bbolt.Tx) error { //nolint:errcheck
		tx.ForEach(func(name []byte, b *bbolt.Bucket) error { //nolint:errcheck
			nodes := process(name, nil, b)
			allNodes = append(allNodes, nodes...)
			return nil
		})
		return nil
	})
	mapNodes(allNodes)
	return allNodes
}

func process(name []byte, path Path, b *bbolt.Bucket) []*TreeNode {
	nodes := []*TreeNode{}
	path = append(path, name)
	node := &TreeNode{
		Path:     path,
		Name:     name,
		IsBucket: true,
	}
	b.ForEach(func(k, v []byte) error { //nolint:errcheck
		if v != nil {
			child := &TreeNode{
				Path:     append(path, k),
				IsBucket: false,
				Name:     k,
				Value:    v,
			}
			node.Children = append(node.Children, child)
		} else {
			nested := b.Bucket(k)
			children := process(k, path, nested)
			node.Children = append(node.Children, children...)
		}
		return nil
	})
	nodes = append(nodes, node)
	return nodes
}

func CreateBucket(path Path) (*bbolt.Bucket, error) {
	log.Println("Create Bucket", pathToString(path))
	var bucket *bbolt.Bucket
	if path == nil {
		return nil, errInvalidPath
	}
	err := db.Update(func(tx *bbolt.Tx) error {
		var err error
		bucket, err = tx.CreateBucketIfNotExists(path[0])
		if err != nil {
			return err
		}
		for _, p := range path[1:] {
			bucket, err = bucket.CreateBucketIfNotExists(p)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return bucket, err
}

func createBucket(path Path, tx *bbolt.Tx) (*bbolt.Bucket, error) {
	bucket, err := tx.CreateBucketIfNotExists(path[0])
	if err != nil {
		return bucket, err
	}
	for _, p := range path[1:] {
		bucket, err = bucket.CreateBucketIfNotExists(p)
		if err != nil {
			return bucket, err
		}
	}
	return bucket, nil
}

func CreateKey(name string, value []byte, path Path) error {
	log.Println("Create Key", pathToString(path), name, string(value))
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := createBucket(path, tx)
		if err != nil {
			return err
		}
		if bucket.Get([]byte(name)) != nil {
			return errKeyExists
		}
		return bucket.Put([]byte(name), value)
	})
}

func DeleteBucket(path Path) error {
	log.Println("Delete bucket", path)
	if path == nil {
		return errInvalidPath
	}
	return db.Update(func(tx *bbolt.Tx) error {
		return deleteBucket(path, tx)
	})
}

func deleteBucket(path Path, tx *bbolt.Tx) error {
	log.Println("delete bucket", path)
	name := path[len(path)-1]
	parent, err := getParentBucket(path, tx)
	if err != nil {
		return err
	}
	if parent == nil {
		return tx.DeleteBucket(name)
	}
	return parent.DeleteBucket(name)
}

func EmptyBucket(path Path) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := getBucket(path, tx)
		if err != nil {
			return err
		}
		if err := bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				return bucket.DeleteBucket(k)
			}
			return bucket.Delete(k)
		}); err != nil {
			return err
		}
		return nil
	})
}

func RenameItem(path Path, name string, isBucket bool) error {
	if isBucket {
		return RenameBucket(path, name)
	}
	return RenameKey(path, name)
}

func RenameBucket(path Path, newName string) error {
	currentName := path[len(path)-1]
	newBucket := &bbolt.Bucket{}
	oldBucket := &bbolt.Bucket{}
	return db.Update(func(tx *bbolt.Tx) error {
		parent, err := getParentBucket(path, tx)
		if err != nil {
			return err
		}
		if parent == nil {
			newBucket, err = tx.CreateBucket([]byte(newName))
			if err != nil {
				return err
			}
			oldBucket = tx.Bucket(currentName)
		} else {
			newBucket, err = parent.CreateBucket([]byte(newName))
			if err != nil {
				return err
			}
			oldBucket = parent.Bucket(currentName)
		}
		err = oldBucket.ForEach(func(k, v []byte) error {
			return newBucket.Put(k, v)
		})
		if err != nil {
			return err
		}
		if parent == nil {
			return tx.DeleteBucket(currentName)
		}
		return parent.DeleteBucket(currentName)
	})
}

func RenameKey(path Path, newName string) error {
	currentName := path[len(path)-1]
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := getParentBucket(path, tx)
		if err != nil {
			return err
		}
		existing := bucket.Get([]byte(newName))
		if existing != nil {
			return errKeyExists
		}
		key := bucket.Get(currentName)
		if key == nil {
			return errInvalidPath
		}
		if err := bucket.Put([]byte(newName), key); err != nil {
			return err
		}
		return bucket.Delete(currentName)
	})
}

func DeleteItem(n *TreeNode) error {
	if n.IsBucket {
		return DeleteBucket(n.Path)
	}
	return DeleteKey(n.Path)
}

func DeleteKey(path Path) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := getParentBucket(path, tx)
		if err != nil {
			return err
		}
		return bucket.Delete(path[len(path)-1])
	})
}

func getParentBucket(path Path, tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if len(path) == 1 {
		// parent is root
		return nil, nil //nolint:nilnil
	}
	return getBucket(path[:len(path)-1], tx)
}

func getBucket(path Path, tx *bbolt.Tx) (*bbolt.Bucket, error) {
	bucket := tx.Bucket(path[0])
	if bucket == nil {
		return &bbolt.Bucket{}, errInvalidPath
	}
	for _, p := range path[1:] {
		bucket = bucket.Bucket(p)
	}
	if bucket == nil {
		return &bbolt.Bucket{}, errInvalidPath
	}
	return bucket, nil
}

func CopyBucket(old, new Path) error {
	log.Println("copy bucket", old, new)
	bucketName := old[len(old)-1]
	return db.Update(func(tx *bbolt.Tx) error {
		if len(old) == 1 {
			bucket := tx.Bucket(bucketName)
			return copyBucket(bucket, new, tx)
		}
		bucket, err := getBucket(old, tx)
		if err != nil {
			return err
		}
		return copyBucket(bucket, new, tx)
	})
}

func MoveBucket(old, new Path) error {
	log.Println("move bucket", old, new)
	bucketName := old[len(old)-1]
	return db.Update(func(tx *bbolt.Tx) error {
		if len(old) == 1 {
			bucket := tx.Bucket(bucketName)
			if err := copyBucket(bucket, new, tx); err != nil {
				return err
			}
			return deleteBucket(old, tx)
		}
		bucket, err := getBucket(old, tx)
		if err != nil {
			return err
		}
		if err := copyBucket(bucket, new, tx); err != nil {
			return err
		}
		return deleteBucket(old, tx)
	})
}

func copyBucket(bucket *bbolt.Bucket, path Path, tx *bbolt.Tx) error {
	newBucket := &bbolt.Bucket{}
	name := path[len(path)-1]
	parent, err := getParentBucket(path, tx)
	if err != nil {
		return err
	}
	// target is root bucket
	if parent == nil {
		newBucket, err = tx.CreateBucket(name)
		if err != nil {
			return err
		}
	} else {
		newBucket, err = parent.CreateBucket(name)
		if err != nil {
			return err
		}
	}
	return bucket.ForEach(func(k, v []byte) error {
		if v == nil {
			newpath := append(path, k)
			nested := bucket.Bucket(k)
			return copyBucket(nested, newpath, tx)
		}
		return newBucket.Put(k, v)
	})
}

func MoveKey(old, new Path) error {
	keyName := old[len(old)-1]
	newName := new[len(new)-1]
	return db.Update(func(tx *bbolt.Tx) error {
		parent, err := getParentBucket(old, tx)
		if err != nil {
			return err
		}
		oldValue := parent.Get(keyName)
		if oldValue == nil {
			return errInvalidPath
		}
		newParent, err := createBucket(new[:len(new)-1], tx)
		if err != nil {
			return err
		}
		// check if key exists
		v := newParent.Get(newName)
		if v != nil {
			return errKeyExists
		}
		if err := newParent.Put(newName, v); err != nil {
			return err
		}
		return parent.Delete(keyName)
	})
}

func CopyKey(current, new Path) error {
	currentName := current[len(current)-1]
	newName := new[len(new)-1]
	return db.Update(func(tx *bbolt.Tx) error {
		parent, err := getParentBucket(current, tx)
		if err != nil {
			return err
		}
		value := parent.Get(currentName)
		if value == nil {
			return errInvalidPath
		}
		newParent, err := createBucket(new[:len(new)-1], tx)
		if err != nil {
			return err
		}
		exists := parent.Get(newName)
		if exists != nil {
			return errKeyExists
		}
		return newParent.Put(newName, value)
	})
}

func UpdateKey(node TreeNode, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		parent, err := getParentBucket(node.Path, tx)
		if err != nil {
			return err
		}
		return parent.Put(node.Name, value)
	})
}
