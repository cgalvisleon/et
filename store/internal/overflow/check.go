package overflow

import (
	"fmt"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

/**
* Pages recorre las páginas de la cadena ref de la versión de owner sin copiar el valor ni cambiar nada (Verify del
* motor): cada una tiene que ser de desborde, de owner (pd_prune_xid), estar en su posición y tener su trozo. Con
* partial (una versión «liberándose», cuya cola ya pudo volver a la lista) termina sin error en la primera página que
* ya no es de owner; si no, esa página es ErrBadChain. Fija una página a la vez con buffer.Ring y nunca visita más
* páginas de las que tiene el valor.
* @param file smgr.FileNo, ref Ref, owner page.XID, partial bool
* @param visit func(block page.BlockNumber) error: cada página de la versión, en orden; un error corta el recorrido
* @return error: ErrBadRef, ErrBadChain, el de visit o el de la caché
**/
func (m *Manager) Pages(file smgr.FileNo, ref Ref, owner page.XID, partial bool, visit func(block page.BlockNumber) error) error {
	if err := checkRef(file, ref); err != nil {
		return err
	}
	if owner == 0 {
		return fmt.Errorf(MSG_BAD_REF, ErrBadRef, file, ref.First, ref.Len)
	}
	block := ref.First
	for seq := 0; seq < pagesOf(ref.Len); seq++ {
		next, owned, err := m.ownPage(file, block, seq, ref.Len, owner, buffer.Ring)
		switch {
		case err != nil:
			return err
		case !owned && partial:
			return nil
		case !owned:
			return badChain(file, block, seq, "page is not part of the version's chain")
		}
		if err := visit(block); err != nil {
			return err
		}
		block = next
	}
	return nil
}
