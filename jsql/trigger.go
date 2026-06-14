package jsql

import (
	"sync"

	"github.com/cgalvisleon/et/et"
)

/**
* defaultTrigger: Returns a Model with default trigger functions for enforcing unique indexes.
* @param model *Model
* @return *Model
**/
func defaultTrigger(model *Model) *Model {
	model.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		var results sync.Map
		var hasError error
		var wg sync.WaitGroup
		for _, validate := range model.Unique {
			wg.Add(1)
			go func(field string) {
				defer wg.Done()

				if hasError != nil {
					return
				}

				val := new.Str(field)
				if len(val) == 0 {
					results.Store(field, false)
					return
				}

				exists, err := model.
					Where(Eq(field, val)).
					Exists()
				if err != nil {
					hasError = err
					return
				}
				results.Store(field, exists)
			}(validate.Name)
		}
		wg.Wait()

		results.Range(func(key, value interface{}) bool {
			if ok, _ := value.(bool); ok {
				return false
			}
			return true
		})

		return hasError
	})

	model.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		var results sync.Map
		var hasError error
		var wg sync.WaitGroup
		for _, validate := range model.Unique {
			wg.Add(1)
			go func(field string) {
				defer wg.Done()

				if hasError != nil {
					return
				}

				newVal := new[field]
				chage := old.IsDeferent(field, newVal)
				if !chage {
					results.Store(field, false)
					return
				}

				ql := model.Where(Eq(field, newVal))
				for _, pk := range model.PrimaryKeys {
					val := old[pk.Name]
					ql = ql.And(Neg(pk.Name, val))
				}

				exists, err := ql.Exists()
				if err != nil {
					hasError = err
					return
				}
				results.Store(field, exists)
			}(validate.Name)
		}
		wg.Wait()

		results.Range(func(key, value interface{}) bool {
			if ok, _ := value.(bool); ok {
				return false
			}
			return true
		})

		return hasError
	})

	return model
}
