package mongo

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/**
* insert
* @param collection string
* @param key string
* @param data et.Json
* @return error
**/
func insert(collection, key string, data et.Json) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	data.Set("_id", key)
	coll := conn.db.Database(conn.dbname).Collection(collection)
	_, err := coll.InsertOne(conn.ctx, data)
	if err != nil {
		return logs.Alert(err)
	}

	return nil
}

/**
* update
* @param collection string
* @param key string
* @param data et.Json
* @return error
**/
func update(collection, key string, data et.Json) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	filter := et.Json{
		"_id": key,
	}
	update := et.Json{
		"$set": data,
	}
	_, err := conn.db.Database(conn.dbname).Collection(collection).UpdateOne(conn.ctx, filter, update)
	if err != nil {
		return logs.Alert(err)
	}

	return nil
}

/**
* Get data from a collection
* @param collection string
* @param key string
* @return et.Items
* @return error
**/
func Get(collection, key string) (et.Item, error) {
	if conn == nil {
		return et.Item{}, logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	findFilter := et.Json{
		"_id": key,
	}
	findOptions := options.Find()
	findOptions.SetMax(1)
	findOptions.SetSkip(0)
	cursor, err := conn.db.Database(conn.dbname).Collection(collection).Find(conn.ctx, findFilter, findOptions)
	if err != nil {
		return et.Item{}, err
	}
	defer cursor.Close(conn.ctx)

	var items et.Item = et.Item{}
	if cursor.Next(conn.ctx) {
		result := et.Json{}
		err := cursor.Decode(&result)
		if err != nil {
			return et.Item{}, err
		}

		items.Ok = true
		items.Result = result
	}

	return items, nil
}

/**
* Insert data into a collection
* @param collection string
* @param key string
* @param data interface{}
* @return error
**/
func Insert(collection, key string, data interface{}) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	switch v := data.(type) {
	case et.Json:
		return insert(collection, key, v)
	case et.Item:
		return insert(collection, key, v.Result)
	case et.Items:
		for _, item := range v.Result {
			k := item.Key("_id")
			err := insert(collection, k, item)
			if err != nil {
				return logs.Alert(err)
			}
		}
	default:
		return logs.Alertm(ERR_NOT_JSON)
	}

	return nil
}

/**
* Set data into a collection
* @param collection string
* @param key string
* @param data interface{}
* @return error
**/
func Set(collection string, data interface{}) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	switch v := data.(type) {
	case et.Json:
		key := v.Key("_id")
		item, err := Get(collection, key)
		if err != nil {
			return logs.Alert(err)
		}

		if !item.Ok {
			return Insert(collection, key, data)
		}

		return update(collection, key, v)
	case et.Item:
		key := v.Key("_id")
		item, err := Get(collection, key)
		if err != nil {
			return logs.Alert(err)
		}

		if !item.Ok {
			return Insert(collection, key, v.Result)
		}

		return update(collection, key, v.Result)
	case et.Items:
		for _, result := range v.Result {
			key := result.Key("_id")
			item, err := Get(collection, key)
			if err != nil {
				return logs.Alert(err)
			}

			if !item.Ok {
				err := Insert(collection, key, result)
				if err != nil {
					return logs.Alert(err)
				}
			} else {
				err := update(collection, key, result)
				if err != nil {
					return logs.Alert(err)
				}
			}
		}
	default:
		return logs.Alertm(ERR_NOT_JSON)
	}

	return nil
}

/**
* Delete data from a collection
* @param collection string
* @param key string
* @return error
**/
func Delete(collection, key string) error {
	if conn == nil {
		return logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	filter := et.Json{
		"_id": key,
	}
	_, err := conn.db.Database(conn.dbname).Collection(collection).DeleteOne(conn.ctx, filter)
	if err != nil {
		return logs.Alert(err)
	}

	return nil
}

/**
* Find data in a collection
* @param collection string
* @param query et.Json
* @param page int
* @param rows int
* @return et.List
* @return error
**/
func Find(collection string, query et.Json, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, logs.Alertm(ERR_NOT_MONGO_SERVICE)
	}

	all, err := conn.db.Database(conn.dbname).Collection(collection).CountDocuments(conn.ctx, query)
	if err != nil {
		return et.List{}, err
	}

	offset := (page - 1) * rows
	findOptions := options.Find()
	findOptions.SetMax(rows)
	findOptions.SetSkip(int64(offset))
	cursor, err := conn.db.Database(conn.dbname).Collection(collection).Find(conn.ctx, query, findOptions)
	if err != nil {
		return et.List{}, err
	}
	defer cursor.Close(conn.ctx)

	var items et.Items = et.Items{}
	for cursor.Next(conn.ctx) {
		result := et.Json{}
		err := cursor.Decode(&result)
		if err != nil {
			return et.List{}, err
		}

		items.Ok = true
		items.Result = append(items.Result, result)
		items.Count++
	}

	return items.ToList(int(all), page, rows), nil
}
