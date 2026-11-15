# features — Mesa de trabajo

El seguimiento de lo que se aprende y lo que se construye, en un solo sistema: los **temas** de aprendizaje, las
**ideas** que salen de ellos y los **features** en que se trabaja (objetivo, plan de tareas, preguntas por decidir, lo
ejecutado y las decisiones). El ciclo es continuo: tema → idea → feature.

- `data/<id>.json`: la fuente, un archivo por feature o por tema. Es lo único que se edita (a mano, por Claude o desde la
  app local `tools/features-hub`).
- `node features/build.mjs` (Node, sin dependencias): genera `index.html` (resumen, **Aprendiendo** —los temas no
  dominados ni archivados, por área—, **Ideas por convertir** y **Features**), `<id>.html` (una página por feature o por
  tema), `todos.html` (el acumulado de ambos tipos) e `ideas.html` (todas las ideas, por proyecto destino). No edites el
  HTML a mano.

## El tipo

Cada JSON lleva `type`: `"feature"` o `"tema"`. Un JSON **sin `type` es un feature** (los de antes siguen igual).

**Número:** `build.mjs` le da a cada feature (`F-1`, `F-2`…) y a cada tema (`T-1`…) un número fijo en `num`, por carpeta y en el orden de `started`, y lo escribe en su JSON. Se muestra antes del título en las páginas, en el hub y en la Mesa de claude.ai. No lo pongas a mano ni lo reutilices: uno nuevo toma el mayor + 1.

## Feature

```json
{
  "type": "feature",
  "id": "mi-feature", "title": "…", "project": "core-studio/api",
  "status": "planeado | listo | en-curso | bloqueado | hecho | descartado",
  "started": "2026-11-14", "updated": "2026-11-14",
  "summary": "una línea para el índice",
  "objective": ["párrafo", "…"],
  "plan": [{"task": "…", "status": "pendiente | en-curso | bloqueado | hecho | descartado | planeado", "note": "opcional"}],
  "questions": [{
    "id": "forma", "question": "¿…?",
    "options": [{"label": "Opción A", "description": "…"}, {"label": "Opción B", "description": "…"}],
    "recommended": 0,
    "status": "abierta | decidida",
    "answer": "el label elegido o texto libre", "note": "comentario opcional del usuario", "decided_at": "2026-11-14"
  }],
  "comments": [{"date": "2026-11-14", "author": "usuario | claude", "text": "…"}],
  "log": [{"date": "2026-11-14", "text": "lo que se hizo"}],
  "decisions": ["…"],
  "refs": [{"path": "spec.md", "label": "EAM-9"}],
  "origin": "opcional: <proyecto>/features/data/<tema>.json#<idea> si viene de una idea"
}
```

- `questions`: las decisiones abiertas del feature. `recommended` es el índice de la opción que recomienda Claude. Al
  responder, la pregunta pasa a `decidida` con `answer`, `decided_at` y, si la hay, `note`; se agrega una entrada al `log`.
- `comments`: la conversación sobre el feature entre el usuario y Claude.
- Estado `listo`: consolidado y listo para ejecutarse. **Pasa a `listo` solo cuando no tiene preguntas abiertas y el
  usuario lo marca**; Claude lanza la ejecución de un feature `listo` (y lo pasa a `en-curso`), por ejemplo con el botón
  «Ejecutar con Claude» del hub. El índice ordena: en curso, bloqueado, listo, planeado, hecho, descartado.
- Avance: tareas `hecho` sobre las que no están `descartado`.

## Tema

```json
{
  "type": "tema",
  "id": "mi-tema", "title": "…", "area": "Bases de datos",
  "status": "por-explorar | estudiando | profundizando | dominado | archivado",
  "started": "2026-11-14", "updated": "2026-11-14",
  "summary": "una línea",
  "why": ["por qué interesa y qué se quiere poder hacer"],
  "questions": [{"id": "…", "question": "¿…?", "status": "abierta | decidida", "answer": "texto libre", "decided_at": "…"}],
  "path": [{"step": "…", "status": "pendiente | en-curso | hecho | descartado", "note": "opcional"}],
  "resources": [{"title": "…", "kind": "libro | artículo | video | código | lección", "ref": "URL o ruta del workspace", "status": "pendiente | leído", "note": "opcional"}],
  "notes": [{"date": "2026-11-14", "text": "lo aprendido o probado"}],
  "experiments": [{"title": "…", "path": "ruta", "result": "…", "date": "2026-11-14"}],
  "ideas": [{"id": "…", "title": "…", "kind": "proyecto | mejora", "target": "proyecto destino, o vacío", "description": "…",
             "status": "idea | propuesta | convertida | descartada", "feature": "<proyecto>/features/data/<id>.json"}],
  "related": ["otro-tema"],
  "comments": [], "refs": []
}
```

- Las preguntas de un tema tienen el mismo formato que las de un feature, pero pueden no tener `options`: se responden
  con texto libre (la página las muestra como «Respondida»).
- `notes` es el diario (las recientes primero): solo lo realmente aprendido o probado, con fecha absoluta.
- `resources.ref`: una URL o una ruta del workspace (`et/learn/x.html`); la página la enlaza con la ruta relativa
  correcta desde esta carpeta (busca junto al proyecto y luego hacia arriba). Una URL no comprobada se dice en `note`.
- Avance: pasos `hecho` (sin los `descartado`) más preguntas respondidas, sobre el total de pasos y preguntas.

## Ideas → features

Una idea nace en un tema (`status: idea`). Solo el usuario la pasa a `propuesta` o la convierte. Convertirla crea un
feature `planeado` en `<target>/features/data/` (en esta misma carpeta si `target` está vacío; si el destino no tiene
`features/`, se crea con estos archivos base) con el `objective` sacado de la idea y del `why` del tema, `refs` al tema,
`origin` = `<proyecto>/features/data/<tema>.json#<idea>` y una primera entrada de `log` que dice de dónde viene. La
idea queda `convertida` con `feature` apuntando al JSON nuevo.

Los textos aceptan `código` y **negrita**.

## En et

- Las rutas de `refs` son relativas a la raíz de `et` (también valen las del workspace).
- Lo que no sale del código, la documentación o el historial se marca «pendiente de confirmar» en la nota de la tarea.
- Las lecciones de `learn/` se enlazan desde los temas de la carpeta `features/` de la raíz del workspace.
