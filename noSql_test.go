package fscache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var noSqlTestCases = []interface{}{
	struct {
		Name string
		Age  int
	}{
		Name: "Jane Doe",
		Age:  25,
	},
	map[string]interface{}{
		"name":    "John Doe",
		"age":     35,
		"colName": "users",
	},
	map[string]interface{}{
		"name":    "Jane Dice",
		"age":     35,
		"colName": "users",
	},
}

func Test_Collection(t *testing.T) {
	ch := Cache{}

	col := ch.NoSql().Collection("user")
	assert.NotNil(t, col)
	assert.Equal(t, "users", col.collectionName)
}

func Test_Insert(t *testing.T) {
	ch := Cache{}

	var counter int
	name := fmt.Sprintf("testCase_%v", counter+1)
	for _, v := range noSqlTestCases {
		t.Run(name, func(t *testing.T) {
			res, err := ch.NoSql().Collection("user").Insert(v)
			if err != nil {
				assert.Error(t, err)
			}

			assert.NotNil(t, v, res)
		})

		counter++
	}
}

func Test_InsertMany(t *testing.T) {
	ch := Cache{}

	err := ch.NoSql().Collection("user").InsertMany(noSqlTestCases)
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
}

func Test_Find(t *testing.T) {
	ch := Cache{}

	// insert a new record
	err := ch.NoSql().Collection("user").InsertMany(noSqlTestCases)
	if err != nil {
		assert.Error(t, err)
	}
	assert.NoError(t, err)

	// filter out record of age 35
	filter := map[string]interface{}{
		"age": 35.0,
	}

	result, err := ch.NoSql().Collection("users").Find(filter).First()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NotNil(t, result)
}

func Test_All(t *testing.T) {
	ch := Cache{}

	// insert a new record
	err := ch.NoSql().Collection("user").InsertMany(noSqlTestCases)
	if err != nil {
		assert.Error(t, err)
	}
	assert.NoError(t, err)

	// filter out records of age 35
	filter := map[string]interface{}{
		"age": 35.0,
	}

	result, err := ch.NoSql().Collection("users").Find(filter).All()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NotNil(t, result)
}
