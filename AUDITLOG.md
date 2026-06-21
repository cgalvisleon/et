# AuditLog.md

Prompt reutilizable para replicar el patrón **AuditLog + onSave/onDelete** en otras structs del repo. Implementación de referencia: `jwf/step.go` (`Step`), replicada igual en `jwf/flow.go` (`Flow`).

## Cuándo usar este patrón

Cuando una struct necesita: historial de auditoría por usuario/acción, hooks configurables que se disparan al guardar/eliminar (p. ej. publicar un evento NATS), y un punto único de inicialización de sus campos privados al crearla o cargarla desde un `Store`.

## Plantilla

Sustituye `{Class}` (nombre de la struct), `{var}` (receptor en minúscula), `{Parent}` (struct contenedora que la crea/carga), `{package}` (paquete destino), `{collection}` (nombre de colección usado en `Store.Set`/`Get`/`Delete`) y `{PACKAGE}` (prefijo de las constantes de error en `msg.go`).

### 1. Campos nuevos en la struct `{Class}`

```go
AuditLog  []et.Json                            `json:"audit_log"`
isDebug   bool                                 `json:"-"`
isChanged bool                                 `json:"-"`
store     Store                                `json:"-"`
onSave    []func({var} *{Class}) error          `json:"-"`
onDelete  []func({var} *{Class}) error          `json:"-"`
```

Omite `store`/`isDebug` si `{Class}` ya los reutiliza por composición desde `{Parent}`.

### 2. Método `up` — inicializa propiedades privadas y registra hooks por defecto

```go
func (s *{Class}) up(parent *{Parent}) *{Class} {
    s.store = parent.store
    s.isDebug = parent.isDebug
    s.onSave = make([]func({var} *{Class}) error, 0)
    s.onDelete = make([]func({var} *{Class}) error, 0)
    s.OnSave(func({var} *{Class}) error {
        key := fmt.Sprintf("{collection}:%s", {var}.ID)
        event.Publish(key, {var}.ToJson())
        return nil
    })
    s.OnDelete(func({var} *{Class}) error {
        key := fmt.Sprintf("{collection}:%s:delete", {var}.ID)
        event.Publish(key, et.Json{"id": {var}.ID})
        return nil
    })
    parent.add{Class}(s) // o el método equivalente del padre que registra la instancia
    return s
}
```

### 3. Registradores de hooks

```go
func (s *{Class}) OnSave(fn func({var} *{Class}) error) *{Class} {
    if s.onSave == nil {
        s.onSave = make([]func({var} *{Class}) error, 0)
    }
    s.onSave = append(s.onSave, fn)
    return s
}

func (s *{Class}) OnDelete(fn func({var} *{Class}) error) *{Class} {
    if s.onDelete == nil {
        s.onDelete = make([]func({var} *{Class}) error, 0)
    }
    s.onDelete = append(s.onDelete, fn)
    return s
}
```

### 4. `addAuditLog` — idéntico en todas las clases, solo cambia el receptor

```go
func (s *{Class}) addAuditLog(userId string, action string) {
    if s.AuditLog == nil {
        s.AuditLog = make([]et.Json, 0)
    }
    now := timezone.Now()
    s.AuditLog = append(s.AuditLog, et.Json{
        "created_at": now,
        "user_id":    userId,
        "action":     action,
    })
    maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
    if len(s.AuditLog) > maxAuditLog {
        s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
    }
    s.isChanged = true
}
```

### 5. `save` — persiste vía `store.Set` y dispara los hooks `onSave`

```go
func (s *{Class}) save() error {
    if s.store == nil {
        return errors.New(MSG_{PACKAGE}_STORE_IS_NIL)
    }
    s.isChanged = false
    if s.isDebug {
        logs.Log(packageName, "save:", s.ToString())
    }
    err := s.store.Set("{collection}", s.ID, s.TenantId, /* ownerId */ "", s)
    if err != nil {
        return err
    }
    for _, onSave := range s.onSave {
        if err := onSave(s); err != nil {
            return err
        }
    }
    return nil
}
```

### 6. Borrado

Se implementa en el método `delete{Class}` del `{Parent}` (no en `{Class}` mismo — así lo hace `Step` vía `WorkFlow.deleteStep`): después de `store.Delete(...)`, recorrer `{var}.onDelete` e invocar cada hook.

```go
func (s *{Parent}) delete{Class}(id string) error {
    if s.store == nil {
        return errors.New(MSG_{PACKAGE}_STORE_IS_NIL)
    }

    {var}, exists := s.get{Class}(id)
    if !exists {
        return Errr{Class}NotFound
    }

    err := s.store.Delete("{collection}", id)
    if err != nil {
        return err
    }

    for _, onDelete := range {var}.onDelete {
        if err := onDelete({var}); err != nil {
            return err
        }
    }

    return nil
}
```

### 7. Constructores: crear (`New`) y cargar (`Load`)

Hay dos variantes según si `{Class}` es una **entidad raíz** (no tiene contenedor en memoria, como `WorkFlow`) o una **entidad hija** indexada dentro de un `{Parent}` (como `Step`/`Flow` dentro de `WorkFlow`). Ambas terminan llamando a `up(...)` de la sección 2.

**Variante A — entidad raíz, funciones de paquete** (referencia: `jwf.New`/`jwf.Load` en `jwf/workflow.go`):

```go
func New(tenantId string, store Store) (*{Class}, error) {
    // si {package} depende de infraestructura, inicialízala aquí, p. ej.:
    // if err := cache.Load(); err != nil { return nil, err }
    // if err := event.Load(); err != nil { return nil, err }

    now := timezone.Now()
    result := &{Class}{
        CreatedAt: now,
        UpdatedAt: now,
        TenantId:  tenantId,
        ID:        reg.ULID(),
        AuditLog:  make([]et.Json, 0),
        // ... resto de campos propios
    }
    return result.up(store)
}

func Load(id string, store Store) (*{Class}, error) {
    if store == nil {
        return nil, errors.New(MSG_{PACKAGE}_STORE_IS_NIL)
    }

    var def et.Json
    exists, err := store.Get("{collection}", id, &def)
    if err != nil {
        return nil, err
    }

    if !exists {
        return nil, errors.New(MSG_{PACKAGE}_NOT_FOUND)
    }

    result := &{Class}{}
    err = json.Unmarshal([]byte(def.ToString()), &result)
    if err != nil {
        return nil, err
    }

    return result.up(store)
}
```

Nota: en esta variante `up` devuelve `(*{Class}, error)` en vez de solo `*{Class}` — `WorkFlow.up` lo usa para recargar recursivamente sus `Flows`/`Steps` hijos y puede fallar. Ajusta la firma de `up` (sección 2) si `{Class}` no necesita propagar errores.

**Variante B — entidad hija, métodos privados en el padre** (referencia: `WorkFlow.newStep`/`loadStep` y `WorkFlow.newFlow`/`loadFlow`):

```go
func (s *{Parent}) new{Class}(/* params propios */, userId string) *{Class} {
    now := timezone.Now()
    result := &{Class}{
        CreatedAt: now,
        UpdatedAt: now,
        TenantId:  s.TenantId,
        ID:        reg.ULID(),
        AuditLog:  make([]et.Json, 0),
        // ... resto de campos propios
    }
    result.addAuditLog(userId, "new_{collection}")
    return result.up(s) // up() ya registra la instancia en el padre vía parent.add{Class}(s)
}

func (s *{Parent}) load{Class}(id string) (*{Class}, error) {
    if s.store == nil {
        return nil, errors.New(MSG_{PACKAGE}_STORE_IS_NIL)
    }

    result, exists := s.get{Class}(id)
    if exists {
        return result, nil // ya está en memoria, evita pegarle al store de nuevo
    }

    exists, err := s.store.Get("{collection}", id, &result)
    if err != nil {
        return nil, err
    }

    if !exists {
        return nil, Errr{Class}NotFound
    }

    return result.up(s), nil
}
```

Usa la variante A si `{Class}` no vive dentro de un mapa de un padre; usa B si `{Parent}` ya expone `add{Class}`/`get{Class}` (sección 2) para indexarla.

## Requisitos previos

- El paquete `{package}` debe definir `MSG_{PACKAGE}_STORE_IS_NIL` (y `Errr{Class}NotFound`, o `MSG_{PACKAGE}_NOT_FOUND` en la variante A) en su `msg.go`.
- `{Parent}` debe exponer `store`, `isDebug` y un método `add{Class}`/`get{Class}` equivalentes a los de `WorkFlow` en `jwf/`.
- Variante A además necesita `reg.ULID()`, `timezone.Now()` y `encoding/json` para el `Unmarshal` de `Load`.

## Candidatos identificados en este repo (sin aplicar todavía)

- `WorkFlow` (`jwf/workflow.go`): **ya implementado** — tiene `AuditLog`/`addAuditLog`, `onSave`/`onDelete`, `OnSave`/`OnDelete`, y `New`/`Load` (variante A) que llaman a `up(store)`.
- `Agent`, `Participant`, `Conversation` (`jia/`): no tienen `AuditLog` ni hooks todavía; ya tienen sus propios `newAgent`/`getAgent` etc. en `jia/ia.go`, así que encajarían en la variante B con `Ia` como `{Parent}`.
- `Tenant`, `App` (`jtenant/`): ya implementan una variante del patrón (incluyendo `NewTenant`/`LoadTenant`, más parecido a la variante A), pero con firma distinta (los callbacks `OnSave`/`OnDelete` reciben `userId` además de la instancia) — revisar si conviene unificar a la firma de `Step`/`Flow` antes de replicar más.
