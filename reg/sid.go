package reg

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
)

/**
* SID: identificador corto, único y ordenable por tiempo. Es un entero de 64 bits
*   [42 bits: ms desde SIDEpoch][10 bits: nodo][12 bits: secuencia]
* escrito como 11 caracteres base62 de ancho fijo, cuyo orden de texto (byte a byte)
* es el mismo que el numérico. Dos generadores con nodos distintos nunca repiten.
**/

const (
	sidNodeBits = 10
	sidSeqBits  = 12
	sidMaxNode  = 1<<sidNodeBits - 1 // 1023
	sidMaxSeq   = 1<<sidSeqBits - 1  // 4095
	sidWidth    = 11                 // 62^11 > 2^64
	sidAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// SIDEpoch: 2024-01-01 UTC en ms; 42 bits alcanzan ~139 años (hasta ~2163).
var SIDEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

/**
* SIDGenerator: Genera SIDs para un nodo; seguro entre goroutines.
**/
type SIDGenerator struct {
	mu   sync.Mutex
	node int64
	last int64 // último ms usado; nunca retrocede
	seq  int64
	now  func() int64
}

/**
* NewSIDGenerator: Crea un generador para el nodo dado.
* @param node int64 (0..1023)
* @return *SIDGenerator, error
**/
func NewSIDGenerator(node int64) (*SIDGenerator, error) {
	if node < 0 || node > sidMaxNode {
		return nil, fmt.Errorf("reg: SID node must be in 0..%d, got %d", sidMaxNode, node)
	}
	return &SIDGenerator{node: node, now: func() int64 { return time.Now().UnixMilli() }}, nil
}

/**
* Node: Retorna el nodo del generador.
* @return int64
**/
func (g *SIDGenerator) Node() int64 {
	return g.node
}

/**
* Next: Retorna el siguiente SID como entero, siempre mayor que el anterior. Si el reloj
* retrocede sigue con el último ms; con más de 4096 ids en un ms toma el ms siguiente.
* @return int64
**/
func (g *SIDGenerator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	ms := g.now() - SIDEpoch
	if ms > g.last {
		g.last, g.seq = ms, 0
	} else {
		g.seq++
		if g.seq > sidMaxSeq {
			g.last++
			g.seq = 0
		}
	}
	return g.last<<(sidNodeBits+sidSeqBits) | g.node<<sidSeqBits | g.seq
}

/**
* NewString: Retorna el siguiente SID como texto de 11 caracteres.
* @return string
**/
func (g *SIDGenerator) NewString() string {
	return EncodeSID(g.Next())
}

/**
* EncodeSID: Convierte un SID a base62 de ancho fijo (11 caracteres).
* @param id int64
* @return string
**/
func EncodeSID(id int64) string {
	var b [sidWidth]byte
	n := uint64(id)
	for i := sidWidth - 1; i >= 0; i-- {
		b[i] = sidAlphabet[n%62]
		n /= 62
	}
	return string(b[:])
}

/**
* DecodeSID: Convierte el texto de un SID a su entero.
* @param s string
* @return int64, error
**/
func DecodeSID(s string) (int64, error) {
	if len(s) != sidWidth {
		return 0, fmt.Errorf("reg: SID must have %d characters, got %d", sidWidth, len(s))
	}
	var n uint64
	for i := range sidWidth {
		d, ok := sidDigit(s[i])
		if !ok {
			return 0, fmt.Errorf("reg: invalid SID character %q", s[i])
		}
		n = n*62 + d
		if n > math.MaxInt64/62 && i < sidWidth-1 {
			return 0, errors.New("reg: SID out of range")
		}
	}
	if n > math.MaxInt64 {
		return 0, errors.New("reg: SID out of range")
	}
	return int64(n), nil
}

/**
* sidDigit: Retorna el valor base62 de c.
* @param c byte
* @return uint64, bool (false si c no es base62)
**/
func sidDigit(c byte) (uint64, bool) {
	switch {
	case c >= '0' && c <= '9':
		return uint64(c - '0'), true
	case c >= 'A' && c <= 'Z':
		return uint64(c-'A') + 10, true
	case c >= 'a' && c <= 'z':
		return uint64(c-'a') + 36, true
	}
	return 0, false
}

/**
* SIDParts: Separa un SID en el instante en que se creó, su nodo y su secuencia.
* @param id int64
* @return time.Time (UTC), int64 (nodo), int64 (secuencia)
**/
func SIDParts(id int64) (time.Time, int64, int64) {
	ms := id>>(sidNodeBits+sidSeqBits) + SIDEpoch
	return time.UnixMilli(ms).UTC(), id >> sidSeqBits & sidMaxNode, id & sidMaxSeq
}

// ---- Generador del proceso ----

var (
	sidOnce sync.Once
	sidMu   sync.Mutex
	sidGen  *SIDGenerator
)

/**
* defaultSID: Retorna el generador del proceso. Su nodo sale de SID_NODE (0..1023); sin
* SID_NODE se elige uno al azar, que puede chocar con el de otro proceso: en producción
* cada proceso debe tener su SID_NODE.
* @return *SIDGenerator
**/
func defaultSID() *SIDGenerator {
	sidOnce.Do(func() {
		node := envar.GetInt64("SID_NODE", -1)
		if node < 0 || node > sidMaxNode {
			node = randomSIDNode()
		}
		gen, _ := NewSIDGenerator(node)
		sidMu.Lock()
		if sidGen == nil {
			sidGen = gen
		}
		sidMu.Unlock()
	})
	sidMu.Lock()
	defer sidMu.Unlock()
	return sidGen
}

/**
* randomSIDNode: Retorna un nodo al azar (crypto/rand).
* @return int64
**/
func randomSIDNode() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(sidMaxNode+1))
	if err != nil {
		return time.Now().UnixNano() & sidMaxNode
	}
	return n.Int64()
}

/**
* SetSIDNode: Fija el nodo del generador del proceso (por ejemplo el número de réplica).
* Los SID ya generados no cambian; los siguientes siguen siendo mayores.
* @param node int64 (0..1023)
* @return error
**/
func SetSIDNode(node int64) error {
	gen, err := NewSIDGenerator(node)
	if err != nil {
		return err
	}
	sidOnce.Do(func() {}) // SID_NODE ya no se lee: manda este nodo
	sidMu.Lock()
	defer sidMu.Unlock()
	if sidGen != nil {
		// Arrancar en el ms siguiente al último usado: con otro nodo, el mismo ms podría
		// dar un SID menor que los ya generados.
		sidGen.mu.Lock()
		gen.last, gen.seq = sidGen.last+1, -1
		sidGen.mu.Unlock()
	}
	sidGen = gen
	return nil
}

/**
* SIDNode: Retorna el nodo del generador del proceso.
* @return int64
**/
func SIDNode() int64 {
	return defaultSID().Node()
}

/**
* SID: Retorna un SID de 11 caracteres del generador del proceso.
* @return string
**/
func SID() string {
	return defaultSID().NewString()
}

/**
* SIDInt: Retorna un SID como entero (para guardarlo como BIGINT).
* @return int64
**/
func SIDInt() int64 {
	return defaultSID().Next()
}

/**
* GenSID: tag:SID
* @param tag string
* @return string
**/
func GenSID(tag string) string {
	return fmt.Sprintf(`%s:%s`, tag, SID())
}

/**
* GetSID: Retorna id, o un SID nuevo si id es "", "*" o "new".
* @param id string
* @return string
**/
func GetSID(id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return SID()
}

/**
* TagSID: Retorna id, o tag:SID nuevo si id es "", "*" o "new".
* @param tag, id string
* @return string
**/
func TagSID(tag, id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return fmt.Sprintf(`%s:%s`, tag, SID())
	}

	return id
}
