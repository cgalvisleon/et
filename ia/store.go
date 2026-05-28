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
			{Name: "id", TypeData: jsql.KEY, Default: ""},
			{Name: "tag", TypeData: jsql.KEY, Default: ""},
			{Name: "owner_id", TypeData: jsql.KEY, Default: ""},
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
* loadAgents
* @return error
**/
func (s *Ia) loadAgents() error {
	var res *Ia
	exists, err := s.store.Get(s.ID, &res)
	if err != nil {
		return err
	}

	if exists && res != nil {
		for k, v := range res.Agents {
			v.up(s)
			s.Agents[k] = v
		}
	}

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
			{Name: "id", TypeData: jsql.KEY, Default: ""},
			{Name: "tag", TypeData: jsql.KEY, Default: ""},
			{Name: "owner_id", TypeData: jsql.KEY, Default: ""},
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
* loadConversations
* @return error
**/
func (s *Ia) loadConversations() error {
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
			{Name: "id", TypeData: jsql.KEY, Default: ""},
			{Name: "tag", TypeData: jsql.KEY, Default: ""},
			{Name: "owner_id", TypeData: jsql.KEY, Default: ""},
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
