package ia

import (
	"github.com/cgalvisleon/et/instances"
	"github.com/cgalvisleon/et/jsql"
)

/**
* initStore
* @return error
**/
func (s *Ia) initStore() error {
	if s.store != nil {
		return nil
	}

	model, err := s.db.Define(jsql.Def{
		Schema:  "ia",
		Name:    "store",
		Version: 1,
		Columns: []jsql.Column{
			{Name: "created_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "updated_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "id", TypeData: jsql.TEXT, Default: ""},
			{Name: "tag", TypeData: jsql.TEXT, Default: ""},
		},
		PrimaryKeys: []jsql.DefIndex{
			{Name: "id", Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "tag", Sorted: true},
		},
		IdxField:    jsql.IDX,
		SourceField: jsql.SOURCE,
	})
	if err != nil {
		return err
	}

	s.store = instances.LoadModel(model)

	return nil
}

/**
* initStoreConversation
* @return error
**/
func (s *Ia) initStoreConversation() error {
	if s.conversationStore != nil {
		return nil
	}

	model, err := s.db.Define(jsql.Def{
		Schema:  "ia",
		Name:    "conversation",
		Version: 1,
		Columns: []jsql.Column{
			{Name: "created_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "updated_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "id", TypeData: jsql.TEXT, Default: ""},
			{Name: "tag", TypeData: jsql.TEXT, Default: ""},
		},
		PrimaryKeys: []jsql.DefIndex{
			{Name: "id", Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "tag", Sorted: true},
		},
		IdxField:    jsql.IDX,
		SourceField: jsql.SOURCE,
	})
	if err != nil {
		return err
	}

	s.conversationStore = instances.LoadModel(model)

	return nil
}

/**
* initStoreMessage
* @return error
**/
func (s *Ia) initStoreMessage() error {
	if s.messageStore != nil {
		return nil
	}

	model, err := s.db.Define(jsql.Def{
		Schema:  "ia",
		Name:    "message",
		Version: 1,
		Columns: []jsql.Column{
			{Name: "created_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "updated_at", TypeData: jsql.DATETIME, Default: ""},
			{Name: "id", TypeData: jsql.TEXT, Default: ""},
			{Name: "tag", TypeData: jsql.TEXT, Default: ""},
		},
		PrimaryKeys: []jsql.DefIndex{
			{Name: "id", Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "tag", Sorted: true},
		},
		IdxField:    jsql.IDX,
		SourceField: jsql.SOURCE,
	})
	if err != nil {
		return err
	}

	s.messageStore = instances.LoadModel(model)

	return nil
}
