package fscache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	// MemgodbStorage storage instance
	MemgodbStorage []interface{}
	// persistMemgodbData to enable persistence of Memgodb data
	persistMemgodbData bool
)

type (
	// Collection object
	Collection struct {
		logger         zerolog.Logger
		collectionName string
	}

	// Insert object implementes One() and Many() to insert new records
	Insert struct {
		obj        interface{}
		collection Collection
	}

	// Filter object implementes One() and All()
	Filter struct {
		objMaps    []map[string]interface{}
		filter     map[string]interface{}
		collection Collection
	}

	// Delete object implementes One() and All()
	Delete struct {
		objMaps    []map[string]interface{}
		filter     map[string]interface{}
		collection Collection
	}

	// Persist objects implemented Persist() used to persist inserted records
	Persist struct {
		Error error
	}

	// Update object implementes One() and All()
	Update struct {
		objMaps    []map[string]interface{}
		filter     map[string]interface{}
		update     map[string]interface{}
		collection Collection
	}
)

// Collection defines the collection(table) name to perform an operations on
func (ns *Memgodb) Collection(col interface{}) *Collection {
	t := reflect.TypeOf(col)

	// run validation
	if reflect.ValueOf(col).IsZero() && col == nil {
		if debug {
			ns.logger.Error().Msg("Collection cannot be empty...")
		}
		panic("Collection cannot be empty...")
	}

	if t.Kind() != reflect.Struct && t.Kind() != reflect.String {
		if debug {
			ns.logger.Error().Msg("Collection must either be a [string] or an [object]")
		}
		panic("Collection must either be a [string] or an [object]")
	}

	var colName string
	if t.Kind() == reflect.Struct {
		colName = strings.ToLower(t.Name())
	} else {
		colName = strings.ToLower(col.(string))
	}

	if len(colName) > 0 && string(colName[len(colName)-1]) != "s" {
		colName = fmt.Sprintf("%ss", colName)
	}

	return &Collection{
		logger:         ns.logger,
		collectionName: colName,
	}
}

// Insert is used to insert a new record into the storage. It has two methods which are One() and Many().
func (c *Collection) Insert(obj interface{}) *Insert {
	return &Insert{
		obj:        obj,
		collection: *c,
	}
}

// One is a method available in Insert(). It adds a new record into the storage with collection name
func (i *Insert) One() (interface{}, error) {
	if i.obj == nil {
		return nil, errors.New("One() params cannot be nil")
	}

	t := reflect.TypeOf(i.obj)

	if t.Kind() != reflect.Struct && t.Kind() != reflect.Map {
		return nil, errors.New("insert() param must either be a [map] or a [struct]")
	}

	objMap, err := i.collection.decode(i.obj)
	if err != nil {
		return nil, err
	}

	objMap["colName"] = i.collection.collectionName
	objMap["id"] = uuid.New()
	objMap["createdAt"] = time.Now()
	objMap["updatedAt"] = nil

	MemgodbStorage = append(MemgodbStorage, objMap)
	return objMap, nil
}

// Many is a method available in Insert(). It adds many records into the storage at once
func (i *Insert) Many(arr interface{}) ([]interface{}, error) {
	if i.obj != nil {
		return nil, errors.New("Many() params must be nil to insert Many")
	}

	t := reflect.TypeOf(arr)

	if t.Kind() != reflect.Slice {
		return nil, errors.New("function param must be a [slice]")
	}

	arrObjs, err := i.collection.decodeMany(arr)
	if err != nil {
		return nil, err
	}

	var savedData []interface{}
	for _, obj := range arrObjs {
		saved, err := i.collection.Insert(obj).One()
		if err != nil {
			return nil, err
		}

		savedData = append(savedData, saved)
	}

	return savedData, nil
}

// FromJsonFile is a method available in Insert(). It adds records into the storage from a json file
func (i *Insert) FromJsonFile(fileLocation string) error {
	if i.obj != nil {
		return errors.New("FromFile() params must be nil to insert from file")
	}

	f, err := os.Open(fileLocation)
	if err != nil {
		return err
	}
	defer f.Close()

	fileByte, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var obj interface{}
	if err := json.Unmarshal(fileByte, &obj); err != nil {
		return errors.New("invalid json file")
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Slice {
		objMap, err := i.collection.decodeMany(obj)
		if err != nil {
			return nil
		}

		_, err = i.collection.Insert(nil).Many(objMap)
		if err != nil {
			return nil
		}
	} else if t.Kind() == reflect.Map {
		objMap, err := i.collection.decode(obj)
		if err != nil {
			return nil
		}

		_, err = i.collection.Insert(objMap).One()
		if err != nil {
			return nil
		}
	} else {
		return errors.New("file must contain either an array of [objects ::: slice] or [object ::: map]")
	}

	return nil
}

// Filter is used to filter records from the storage. It has two methods which are First() and All().
func (c *Collection) Filter(filter map[string]interface{}) *Filter {
	var objMaps []map[string]interface{}
	var err error

	if filter != nil {
		objMaps, err = c.decodeMany(MemgodbStorage)
		if err != nil {
			return nil
		}
	}

	return &Filter{
		objMaps:    objMaps,
		filter:     filter,
		collection: *c,
	}
}

// First is a method available in Filter(), it returns the first matching record from the filter.
func (f *Filter) First() (map[string]interface{}, error) {
	if f.objMaps == nil {
		return nil, errors.New("filter params cannot be nil")
	}

	notFound := true
	var foundObj map[string]interface{}
	counter := 0
	for _, item := range f.objMaps {
		for key, val := range f.filter {
			if item["colName"] == f.collection.collectionName {
				if v, ok := item[key]; ok && val == v {
					if counter < 1 {
						notFound = false
						foundObj = item
						counter++
					}
					break
				}
			}
		}
	}

	if notFound {
		return nil, errors.New("record not found")
	}

	return foundObj, nil
}

// All is a method available in Filter(), it returns all the matching records from the filter.
func (f *Filter) All() ([]map[string]interface{}, error) {
	if f.objMaps == nil {
		var objMaps []map[string]interface{}
		arrObj, err := json.Marshal(MemgodbStorage)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(arrObj, &objMaps); err != nil {
			return nil, err
		}

		return objMaps, nil
	}

	notFound := true
	var foundObj []map[string]interface{}
	for _, item := range f.objMaps {
		for key, val := range f.filter {
			if item["colName"] == f.collection.collectionName {
				if v, ok := item[key]; ok && val == v {
					notFound = false
					foundObj = append(foundObj, item)
				}
			}
		}
	}

	if notFound {
		return nil, errors.New("record not found")
	}

	return foundObj, nil
}

// Delete is used to delete a new record from the storage. It has two methods which are One() and Many().
func (c *Collection) Delete(filter map[string]interface{}) *Delete {
	var objMaps []map[string]interface{}
	var err error

	if filter != nil {
		objMaps, err = c.decodeMany(MemgodbStorage)
		if err != nil {
			return nil
		}
	}

	return &Delete{
		objMaps:    objMaps,
		filter:     filter,
		collection: *c,
	}
}

// One is a method available in Delete(), it deletes a record and returns an error if any.
func (d *Delete) One() error {
	if d.objMaps == nil {
		return errors.New("filter params cannot be nil")
	}

	notFound := true
	for index, item := range d.objMaps {
		for key, val := range d.filter {
			if item["colName"] == d.collection.collectionName {
				if v, ok := item[key]; ok && val == v {
					notFound = false
					if index < (len(MemgodbStorage) - 1) {
						MemgodbStorage = append(MemgodbStorage[:index], MemgodbStorage[index+1:]...)
						index--
						break
					} else {
						MemgodbStorage = MemgodbStorage[:index]
						break
					}
				}
			}
		}
	}

	if notFound {
		return errors.New("record not found")
	}

	return nil
}

// All is a method available in Delete(), it deletes matching records from the filter and returns an error if any.
func (d *Delete) All() error {
	if d.objMaps == nil {
		MemgodbStorage = MemgodbStorage[:0]
		return nil
	}

	notFound := true
	for index, item := range d.objMaps {
		for key, val := range d.filter {
			if item["colName"] == d.collection.collectionName {
				if v, ok := item[key]; ok && val == v {
					notFound = false
					if index < (len(MemgodbStorage) - 1) {
						MemgodbStorage = append(MemgodbStorage[:index], MemgodbStorage[index+1:]...)
						index--
					} else {
						MemgodbStorage = MemgodbStorage[:index]
					}
				}
			}
		}
	}

	if notFound {
		return errors.New("record not found")
	}

	return nil
}

// Update is used to update a existing record in the storage. It has a method which is One().
func (c *Collection) Update(filter, obj map[string]interface{}) *Update {
	var objMaps []map[string]interface{}
	var err error

	if filter != nil {
		objMaps, err = c.decodeMany(MemgodbStorage)
		if err != nil {
			return nil
		}
	}

	return &Update{
		objMaps:    objMaps,
		filter:     filter,
		update:     obj,
		collection: *c,
	}
}

// One is a method available in Update(), it updates matching records from the filter, makes the necessry updated and returns an error if any.
func (u *Update) One() error {
	if u.objMaps == nil {
		return errors.New("filter params cannot be nil")
	}

	notFound := true
	counter := 0
	for index, item := range u.objMaps {
		for key, val := range u.filter {
			if item["colName"] == u.collection.collectionName {
				if v, ok := item[key]; ok && val == v {
					notFound = false
					if counter < 1 {
						for _, updateValue := range u.update {
							item[key] = updateValue
							counter++
							break
						}
						item["updatedAt"] = time.Now()
					}
					MemgodbStorage[index] = item
				}
			}
		}
	}

	if notFound {
		return errors.New("record not found")
	}

	return nil
}

// LoadDefault is used to load datas from the json file saved on the server using Persist() if any.
func (n *Memgodb) LoadDefault() error {
	f, err := os.Open("./memgodbstorage.json")
	if err != nil {
		return errors.New("error finding file")
	}
	defer f.Close()

	fileByte, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var obj interface{}
	if err := json.Unmarshal(fileByte, &obj); err != nil {
		return errors.New("invalid json file")
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Slice {
		var objMap []interface{}
		jsonByte, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(jsonByte, &objMap); err != nil {
			return err
		}

		MemgodbStorage = append(MemgodbStorage, objMap...)
	} else if t.Kind() == reflect.Map {
		var objMap interface{}
		jsonByte, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(jsonByte, &objMap); err != nil {
			return err
		}

		MemgodbStorage = append(MemgodbStorage, objMap)
	}

	return nil
}

// Persist is used to write data to file. All datas will be saved into a json file on the server.

// This method will make sure all your your data's are saved into a json file. A cronJon runs ever minute and writes your data(s) into a json file to ensure data integrity
func (n *Memgodb) Persist() error {
	if MemgodbStorage == nil {
		return nil
	}

	persistMemgodbData = true
	jsonByte, err := json.Marshal(MemgodbStorage)
	if err != nil {
		return err
	}

	file, err := os.Create("./memgodbstorage.json")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonByte)
	if err != nil {
		return err
	}

	return nil
}

// decode decodes an interface{} into a map[string]interface{}
func (c *Collection) decode(obj interface{}) (map[string]interface{}, error) {
	objMap := make(map[string]interface{})
	jsonObj, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonObj, &objMap); err != nil {
		return nil, err
	}

	return objMap, nil
}

// decodeMany decodes an interface{} into an []map[string]interface{}
func (c *Collection) decodeMany(arr interface{}) ([]map[string]interface{}, error) {
	var arrObjs []map[string]interface{}
	arrObj, err := json.Marshal(arr)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(arrObj, &arrObjs); err != nil {
		return nil, err
	}

	return arrObjs, nil
}
