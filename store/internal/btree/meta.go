package btree

import "github.com/cgalvisleon/et/store/internal/page"

// Posiciones de los campos de la página meta (doc.go). La magia, el bloque y la cabeza de la lista de libres los
// define page (page/meta.go), compartidos con fsm.
const (
	offMetaMagic     = page.MetaOffMagic
	offMetaVersion   = page.HeaderSize + 4
	offMetaRoot      = page.HeaderSize + 8
	offMetaLevel     = page.HeaderSize + 12
	offMetaFastRoot  = page.HeaderSize + 16
	offMetaFastLevel = page.HeaderSize + 20
	offMetaFreeHead  = page.MetaOffFreeHead

	// metaVersion es la versión del formato de la meta.
	metaVersion = 1
)

// meta es lo que el árbol lee de la página meta.
type meta struct {
	root  page.BlockNumber
	level uint32
}

/**
* readMetaPage lee la raíz de la página meta, comprobando que lo sea (iniciada, magia, versión, page.OpaqueMeta).
* @param p page.Page
* @return meta, error: ErrCorrupt
**/
func readMetaPage(p page.Page) (meta, error) {
	if p.IsNew() || le.Uint32(p[offMetaMagic:]) != MetaMagic || le.Uint32(p[offMetaVersion:]) != metaVersion {
		return meta{}, corruptAt(MetaBlock, "not a meta page")
	}
	if o, err := p.Opaque(); err != nil || o.Flags&page.OpaqueMeta == 0 {
		return meta{}, corruptAt(MetaBlock, "meta page without OpaqueMeta")
	}
	m := meta{root: page.BlockNumber(le.Uint32(p[offMetaRoot:])), level: le.Uint32(p[offMetaLevel:])}
	if m.root == MetaBlock {
		return meta{}, corruptAt(MetaBlock, "root is the meta page")
	}
	return m, nil
}

/**
* initMetaPage deja p como la meta de un store nuevo: raíz root de nivel 0 y lista de libres vacía. pd_lower cubre
* el contenido para que la imagen de página entera del WAL no lo deje fuera.
* @param p page.Page, root page.BlockNumber
* @return error
**/
func initMetaPage(p page.Page, root page.BlockNumber) error {
	if err := p.Init(page.OpaqueSize); err != nil {
		return err
	}
	if err := p.SetOpaque(page.Opaque{Flags: page.OpaqueMeta}); err != nil {
		return err
	}
	le.PutUint32(p[offMetaMagic:], MetaMagic)
	le.PutUint32(p[offMetaVersion:], metaVersion)
	setMetaRoot(p, root, 0)
	return p.SetLower(page.HeaderSize + MetaSize)
}

/**
* setMetaRoot anota la raíz y su nivel. La raíz rápida es la misma: sin borrado de páginas (fase 2) no hay niveles
* de un solo hijo que saltar.
* @param p page.Page, root page.BlockNumber, level uint32
**/
func setMetaRoot(p page.Page, root page.BlockNumber, level uint32) {
	le.PutUint32(p[offMetaRoot:], uint32(root))
	le.PutUint32(p[offMetaLevel:], level)
	le.PutUint32(p[offMetaFastRoot:], uint32(root))
	le.PutUint32(p[offMetaFastLevel:], level)
}
