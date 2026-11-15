// Mesa de trabajo: genera las páginas de features/ a partir de data/*.json. Cada JSON es un feature (`type: "feature"`,
// o sin `type`) o un tema de aprendizaje (`type: "tema"`). Genera index.html (resumen, Aprendiendo, Ideas por convertir y
// Features), <id>.html (una por feature o tema), todos.html (el acumulado) e ideas.html (las ideas de los temas por destino).
// Uso: node features/build.mjs (desde el proyecto o desde features/). Sin dependencias.
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const FAVICON = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Crect width='64' height='64' rx='14' fill='%232f6feb'/%3E%3Crect x='14' y='14' width='36' height='40' rx='5' fill='%23fff'/%3E%3Crect x='24' y='10' width='16' height='8' rx='3' fill='%23f4c542'/%3E%3Cpath d='M20 28l3 3 6-6M20 42l3 3 6-6' fill='none' stroke='%231a7f37' stroke-width='3' stroke-linecap='round' stroke-linejoin='round'/%3E%3Cpath d='M33 29h11M33 43h11' stroke='%238c959f' stroke-width='3' stroke-linecap='round'/%3E%3C/svg%3E"; // el ícono de la Mesa de trabajo, en línea: sin archivo aparte en cada carpeta

const DIR = dirname(fileURLToPath(import.meta.url));
const DATA = join(DIR, "data");
const ROOT = dirname(DIR); // la carpeta del proyecto (la que contiene features/)

// Las etiquetas de todos los estados (features, tareas, preguntas, temas, pasos, recursos e ideas)
const STATUS = {
  planeado: "Planeado",
  listo: "Listo",
  "en-curso": "En curso",
  bloqueado: "Bloqueado",
  hecho: "Hecho",
  pendiente: "Pendiente",
  descartado: "Descartado",
  abierta: "Abierta",
  decidida: "Decidida",
  "por-explorar": "Por explorar",
  estudiando: "Estudiando",
  profundizando: "Profundizando",
  dominado: "Dominado",
  archivado: "Archivado",
  "leído": "Leído",
  idea: "Idea",
  propuesta: "Propuesta",
  convertida: "Convertida",
  descartada: "Descartada",
};
// listo = consolidado y sin preguntas abiertas: Claude puede lanzar su ejecución
const ORDER = ["en-curso", "bloqueado", "listo", "planeado", "hecho", "descartado"];
const TEMA_ORDER = ["estudiando", "profundizando", "por-explorar", "dominado", "archivado"];
const IDEA_ORDER = ["propuesta", "idea", "convertida", "descartada"];
const CLOSED_TEMA = new Set(["dominado", "archivado"]);

const esc = (s) => String(s ?? "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
// Texto con `código` y **negrita**
const inline = (s) =>
  esc(s)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
// La clase de color de un estado, sin acentos (leído → s-leido)
const cls = (status) => "s-" + String(status ?? "").normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/[^a-z0-9-]/gi, "");
const badge = (status, label) => `<span class="badge ${esc(cls(status))}">${esc(label ?? STATUS[status] ?? status)}</span>`;
const typeOf = (f) => (f.type === "tema" ? "tema" : "feature");
// El número fijo de un feature (F-1, F-2…) o de un tema (T-1…): se asigna una vez en `num` y no se reutiliza
const numOf = (f) => (f.num ? `${typeOf(f) === "tema" ? "T" : "F"}-${f.num}` : "");
const numTag = (f) => (f.num ? `<span class="ref-num">${numOf(f)}</span> ` : "");
const titled = (f) => (f.num ? `${numOf(f)} · ${f.title}` : f.title);

// Una ruta del workspace o del proyecto como enlace relativo desde features/: se busca primero junto al proyecto y luego
// subiendo por las carpetas padre (así sirven `et/learn/x.html` y `docs/x.md`). Devuelve null si no existe.
const localHref = (ref) => {
  if (typeof ref !== "string" || !ref || /^[a-z]+:/i.test(ref) || ref.startsWith("/")) return null;
  const [path, hash] = ref.split("#");
  if (!path || path.split(/[\\/]/).includes("..")) return null;
  for (let dir = ROOT; ; dir = dirname(dir)) {
    const file = resolve(dir, path);
    if (existsSync(file)) return relative(DIR, file).split(sep).join("/") + (hash ? `#${hash}` : "");
    if (dirname(dir) === dir) return null;
  }
};
const isUrl = (ref) => typeof ref === "string" && /^https?:\/\//i.test(ref);
// Un enlace a una ruta del workspace o a una URL; si no se encuentra, la ruta como código
const refLink = (ref, label) => {
  if (isUrl(ref)) return `<a href="${esc(ref)}" target="_blank" rel="noopener noreferrer">${label ? inline(label) : esc(ref)}</a>`;
  const href = localHref(ref);
  if (href) return `<a href="${esc(href)}">${label ? inline(label) : `<code>${esc(ref)}</code>`}</a>`;
  return `${label ? `${inline(label)} ` : ""}<code>${esc(ref)}</code>${ref ? ' <span class="muted note">(no encontrado)</span>' : ""}`;
};
// La página HTML de un JSON de features (…/features/data/<id>.json → …/features/<id>.html)
const featureHref = (ref) => {
  if (typeof ref !== "string" || !ref) return null;
  const [path, hash] = ref.split("#");
  const href = localHref(path);
  if (!href) return null;
  // El ancla de una idea en la página del tema es idea-<tema>-<idea> (así llega el `origin` de un feature convertido)
  const m = /(^|\/)data\/([a-z0-9-]+)\.json$/.exec(href);
  if (!m) return null;
  return href.slice(0, m.index) + `${m[1]}${m[2]}.html` + (hash ? `#idea-${m[2]}-${hash}` : "");
};

// ---------- Features ----------

// Avance: las tareas hechas sobre las que cuentan (las descartadas no cuentan)
const progressFeature = (f) => {
  const tasks = (f.plan ?? []).filter((t) => t.status !== "descartado");
  const done = tasks.filter((t) => t.status === "hecho").length;
  return { done, total: tasks.length };
};
// Avance de un tema: pasos hechos (sin los descartados) más preguntas respondidas, sobre el total
const progressTema = (t) => {
  const steps = (t.path ?? []).filter((s) => s.status !== "descartado");
  const qs = t.questions ?? [];
  const done = steps.filter((s) => s.status === "hecho").length + qs.filter((q) => q.status === "decidida").length;
  return { done, total: steps.length + qs.length };
};
const progress = (f) => (typeOf(f) === "tema" ? progressTema(f) : progressFeature(f));
// Las preguntas abiertas (las que no están decididas)
const openQuestions = (f) => (f.questions ?? []).filter((q) => q.status !== "decidida");
const qbadge = (f) => {
  const n = openQuestions(f).length;
  return n ? `<span class="badge s-abierta" title="Preguntas abiertas">${n} ${n === 1 ? "pregunta" : "preguntas"}</span>` : "";
};
const bar = (f) => {
  const p = progress(f);
  const pct = p.total ? Math.round((p.done * 100) / p.total) : 0;
  const what = typeOf(f) === "tema" ? "pasos y preguntas" : "tareas";
  return `<div class="progress" title="${p.done} de ${p.total} ${what}"><div style="width:${pct}%"></div></div><span class="muted">${p.done}/${p.total}</span>`;
};

const page = (title, body, nav = "") => `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)}</title>
<link rel="icon" type="image/svg+xml" href="${FAVICON}">
<link rel="stylesheet" href="styles.css">
</head>
<body>
<header class="top"><a href="index.html">Mesa de trabajo</a>${nav}<span class="spacer"></span><a href="ideas.html">Ideas</a><a href="todos.html">Acumulado</a></header>
<main>
${body}
</main>
<footer class="muted">Generado con <code>node features/build.mjs</code> desde <code>features/data/*.json</code>; no edites este HTML a mano.</footer>
</body>
</html>
`;

const questionsHtml = (f, h, { tema = false } = {}) => {
  const open = openQuestions(f)
    .map((q) => {
      const opts = (q.options ?? [])
        .map(
          (o, i) =>
            `<li class="${i === q.recommended ? "rec" : ""}"><strong>${inline(o.label)}</strong>${i === q.recommended ? ' <span class="badge s-listo">Recomendada</span>' : ""}${o.description ? `<div class="muted note">${inline(o.description)}</div>` : ""}</li>`,
        )
        .join("\n");
      return `<div class="question open" id="q-${esc(f.id)}-${esc(q.id)}"><div class="q">${badge("abierta")} ${inline(q.question)}</div>${opts ? `<ol class="options">\n${opts}\n</ol>` : ""}${q.note ? `<div class="muted note">${inline(q.note)}</div>` : ""}</div>`;
    })
    .join("\n");
  const decided = (f.questions ?? [])
    .filter((q) => q.status === "decidida")
    .map(
      (q) =>
        `<div class="question decided"><div class="q">${badge("decidida", tema ? "Respondida" : undefined)} ${inline(q.question)}</div><div class="answer">${tema ? inline(q.answer) : `<strong>${inline(q.answer)}</strong>`}${q.decided_at ? ` <span class="muted">· ${esc(q.decided_at)}</span>` : ""}</div>${q.note ? `<div class="muted note">${inline(q.note)}</div>` : ""}</div>`,
    )
    .join("\n");
  return { open, decided };
};
const commentsHtml = (f) =>
  (f.comments ?? [])
    .map((c) => `<li class="by-${esc(c.author)}"><time>${esc(c.date)}</time> <span class="chip">${esc(c.author === "claude" ? "Claude" : "Usuario")}</span> ${inline(c.text)}</li>`)
    .join("\n");

// El contenido de un feature (sus secciones), igual en su página y en el acumulado
const sectionFeature = (f, level = 1) => {
  const h = (n) => `h${Math.min(level + n, 6)}`;
  const meta = [
    badge(f.status),
    qbadge(f),
    f.project ? `<span class="chip">${esc(f.project)}</span>` : "",
    `<span class="muted">Inicio ${esc(f.started)} · Actualizado ${esc(f.updated)}</span>`,
  ].join(" ");
  const objective = (f.objective ?? []).map((p) => `<p>${inline(p)}</p>`).join("\n");
  const plan = (f.plan ?? [])
    .map(
      (t, i) =>
        `<tr><td class="num">${i + 1}</td><td>${inline(t.task)}${t.note ? `<div class="muted note">${inline(t.note)}</div>` : ""}</td><td>${badge(t.status)}</td></tr>`,
    )
    .join("\n");
  const log = [...(f.log ?? [])]
    .sort((a, b) => (a.date < b.date ? 1 : -1))
    .map((e) => `<li><time>${esc(e.date)}</time> ${inline(e.text)}</li>`)
    .join("\n");
  const decisions = (f.decisions ?? []).map((d) => `<li>${inline(d)}</li>`).join("\n");
  const { open, decided } = questionsHtml(f, h);
  const comments = commentsHtml(f);
  const refs = (f.refs ?? []).map((r) => `<li><code>${esc(r.path)}</code>${r.label ? ` — ${inline(r.label)}` : ""}</li>`).join("\n");
  const originHref = featureHref(f.origin);
  const origin = f.origin
    ? `<p class="origin">Viene del tema ${originHref ? `<a href="${esc(originHref)}"><code>${esc(f.origin)}</code></a>` : `<code>${esc(f.origin)}</code>`}</p>`
    : "";
  return `<${h(0)} id="${esc(f.id)}">${numTag(f)}${esc(f.title)}</${h(0)}>
<div class="meta">${meta}</div>
<div class="row">${bar(f)}</div>
${origin}<${h(1)}>Objetivo</${h(1)}>
${objective}
<${h(1)}>Plan de tareas</${h(1)}>
<table class="plan"><thead><tr><th>#</th><th>Tarea</th><th>Estado</th></tr></thead><tbody>
${plan}
</tbody></table>
${open ? `<${h(1)} id="preguntas-${esc(f.id)}">Preguntas abiertas</${h(1)}>\n<p class="muted note">El feature pasa a <em>listo</em> cuando no quedan preguntas abiertas y el usuario lo marca.</p>\n${open}` : ""}
${decided ? `<${h(1)}>Preguntas decididas</${h(1)}>\n${decided}` : ""}
${comments ? `<${h(1)}>Comentarios</${h(1)}>\n<ul class="log comments">\n${comments}\n</ul>` : ""}
<${h(1)}>Tareas ejecutadas</${h(1)}>
<ul class="log">
${log || '<li class="muted">Nada todavía.</li>'}
</ul>
${decisions ? `<${h(1)}>Decisiones</${h(1)}>\n<ul>\n${decisions}\n</ul>` : ""}
${refs ? `<${h(1)}>Referencias</${h(1)}>\n<ul class="refs">\n${refs}\n</ul>` : ""}`;
};

// ---------- Temas ----------

const KIND = { libro: "Libro", "artículo": "Artículo", video: "Video", "código": "Código", "lección": "Lección", proyecto: "Proyecto", mejora: "Mejora" };
const ideaTarget = (i) => i.target || "";
const ideaHtml = (t, i) => {
  const fh = i.feature ? featureHref(i.feature) : null;
  const feature = i.feature ? ` → feature ${fh ? `<a href="${esc(fh)}"><code>${esc(i.feature)}</code></a>` : `<code>${esc(i.feature)}</code>`}` : "";
  return `<li class="idea ${esc(cls(i.status))}" id="idea-${esc(t.id)}-${esc(i.id)}"><div class="meta">${badge(i.status ?? "idea")} <span class="chip">${esc(KIND[i.kind] ?? i.kind ?? "idea")}</span> <strong>${inline(i.title)}</strong> <span class="muted note">destino ${i.target ? `<code>${esc(i.target)}</code>` : "esta carpeta"}</span></div>${i.description ? `<div class="note">${inline(i.description)}</div>` : ""}${feature ? `<div class="note">${feature}</div>` : ""}</li>`;
};

// El contenido de un tema: un cuaderno de estudio
const sectionTema = (t, level = 1, temaIds = new Set()) => {
  const h = (n) => `h${Math.min(level + n, 6)}`;
  const meta = [
    badge(t.status ?? "por-explorar"),
    qbadge(t),
    t.area ? `<span class="chip">${esc(t.area)}</span>` : "",
    `<span class="muted">Inicio ${esc(t.started)} · Actualizado ${esc(t.updated)}</span>`,
  ].join(" ");
  const why = (t.why ?? []).map((p) => `<p>${inline(p)}</p>`).join("\n");
  const { open, decided } = questionsHtml(t, h, { tema: true });
  const path = (t.path ?? [])
    .map(
      (s, i) =>
        `<li class="step ${esc(cls(s.status))}"><span class="num">${i + 1}</span><div>${inline(s.step)}${s.note ? `<div class="muted note">${inline(s.note)}</div>` : ""}</div>${badge(s.status ?? "pendiente")}</li>`,
    )
    .join("\n");
  const resources = (t.resources ?? [])
    .map(
      (r) =>
        `<li><div class="meta"><span class="chip">${esc(KIND[r.kind] ?? r.kind ?? "recurso")}</span> ${refLink(r.ref, r.title)} ${badge(r.status ?? "pendiente")}</div>${r.note ? `<div class="muted note">${inline(r.note)}</div>` : ""}</li>`,
    )
    .join("\n");
  const notes = [...(t.notes ?? [])]
    .sort((a, b) => (a.date < b.date ? 1 : -1))
    .map((n) => `<li><time>${esc(n.date)}</time><div>${inline(n.text)}</div></li>`)
    .join("\n");
  const experiments = (t.experiments ?? [])
    .map(
      (x) =>
        `<li><div class="meta"><strong>${inline(x.title)}</strong>${x.date ? ` <time class="muted">${esc(x.date)}</time>` : ""}${x.path ? ` ${refLink(x.path)}` : ""}</div>${x.result ? `<div class="note">${inline(x.result)}</div>` : ""}</li>`,
    )
    .join("\n");
  const ideas = [...(t.ideas ?? [])].sort((a, b) => IDEA_ORDER.indexOf(a.status) - IDEA_ORDER.indexOf(b.status)).map((i) => ideaHtml(t, i)).join("\n");
  const related = (t.related ?? []).map((id) => `<li>${temaIds.has(id) ? `<a href="${esc(id)}.html">${esc(id)}</a>` : `<code>${esc(id)}</code>`}</li>`).join("\n");
  const comments = commentsHtml(t);
  const refs = (t.refs ?? []).map((r) => `<li>${refLink(r.path)}${r.label ? ` — ${inline(r.label)}` : ""}</li>`).join("\n");
  return `<${h(0)} id="${esc(t.id)}"><span class="kicker">Tema</span>${numTag(t)}${esc(t.title)}</${h(0)}>
<div class="meta">${meta}</div>
<div class="row">${bar(t)}</div>
${t.summary ? `<p class="lead">${inline(t.summary)}</p>` : ""}
<${h(1)}>Por qué</${h(1)}>
${why || '<p class="muted">Sin escribir todavía.</p>'}
${open ? `<${h(1)} id="preguntas-${esc(t.id)}">Preguntas abiertas</${h(1)}>\n${open}` : ""}
<${h(1)}>Ruta</${h(1)}>
${path ? `<ol class="path">\n${path}\n</ol>` : '<p class="muted">Sin pasos todavía.</p>'}
<${h(1)}>Recursos</${h(1)}>
${resources ? `<ul class="resources">\n${resources}\n</ul>` : '<p class="muted">Sin recursos todavía.</p>'}
<${h(1)}>Diario</${h(1)}>
<ul class="diary">
${notes || '<li class="muted">Sin notas todavía.</li>'}
</ul>
${decided ? `<${h(1)}>Preguntas respondidas</${h(1)}>\n${decided}` : ""}
${experiments ? `<${h(1)}>Experimentos</${h(1)}>\n<ul class="experiments">\n${experiments}\n</ul>` : ""}
${ideas ? `<${h(1)}>Ideas</${h(1)}>\n<ul class="ideas">\n${ideas}\n</ul>` : ""}
${related ? `<${h(1)}>Relacionados</${h(1)}>\n<ul>\n${related}\n</ul>` : ""}
${comments ? `<${h(1)}>Comentarios</${h(1)}>\n<ul class="log comments">\n${comments}\n</ul>` : ""}
${refs ? `<${h(1)}>Referencias</${h(1)}>\n<ul class="refs">\n${refs}\n</ul>` : ""}`;
};

// ---------- Carga ----------

const items = readdirSync(DATA)
  .filter((n) => n.endsWith(".json"))
  .map((n) => {
    const f = JSON.parse(readFileSync(join(DATA, n), "utf8"));
    if (!f.id) f.id = n.replace(/\.json$/, "");
    f._file = join(DATA, n);
    return f;
  });
// Numera lo que aún no tiene número: por tipo, en el orden en que empezó, a partir del mayor ya usado; el número
// queda escrito en su JSON (justo después de `id`) para que no cambie aunque se borren otros
for (const type of ["feature", "tema"]) {
  const list = items.filter((f) => typeOf(f) === type);
  let next = Math.max(0, ...list.map((f) => Number(f.num) || 0));
  const fresh = list.filter((f) => !f.num).sort((a, b) => String(a.started ?? "").localeCompare(String(b.started ?? "")) || a.id.localeCompare(b.id));
  for (const f of fresh) {
    f.num = ++next;
    const raw = JSON.parse(readFileSync(f._file, "utf8"));
    const out = {};
    for (const [k, v] of Object.entries(raw)) {
      if (k === "num") continue;
      out[k] = v;
      if (k === "id") out.num = f.num;
    }
    if (!("num" in out)) Object.assign(out, { num: f.num });
    writeFileSync(f._file, JSON.stringify(out, null, 2) + "\n");
  }
}
for (const f of items) delete f._file;
// Igual que antes de los temas: con la misma fecha, el orden de readdir (así el índice de features no cambia)
const byUpdated = (a, b) => (a.updated < b.updated ? 1 : -1);
const features = items.filter((f) => typeOf(f) === "feature").sort((a, b) => ORDER.indexOf(a.status) - ORDER.indexOf(b.status) || byUpdated(a, b));
const temas = items.filter((f) => typeOf(f) === "tema").sort((a, b) => TEMA_ORDER.indexOf(a.status) - TEMA_ORDER.indexOf(b.status) || byUpdated(a, b));
const temaIds = new Set(temas.map((t) => t.id));
const allIdeas = temas.flatMap((t) => (t.ideas ?? []).map((i) => ({ tema: t, idea: i })));
const pendingIdeas = allIdeas.filter((x) => (x.idea.status ?? "idea") === "idea" || x.idea.status === "propuesta");

for (const f of features) {
  writeFileSync(join(DIR, `${f.id}.html`), page(titled(f), `<article>\n${sectionFeature(f)}\n</article>`, `<span class="sep">›</span><span>${esc(titled(f))}</span>`));
}
for (const t of temas) {
  writeFileSync(
    join(DIR, `${t.id}.html`),
    page(titled(t), `<article class="tema">\n${sectionTema(t, 1, temaIds)}\n</article>`, `<span class="sep">›</span><span>Tema</span><span class="sep">›</span><span>${esc(titled(t))}</span>`),
  );
}

// ---------- Resumen ----------

const count = (list, s) => list.filter((f) => f.status === s).length;
const totals = features.reduce(
  (acc, f) => {
    const p = progressFeature(f);
    return { done: acc.done + p.done, total: acc.total + p.total };
  },
  { done: 0, total: 0 },
);
const openTotal = items.reduce((n, f) => n + openQuestions(f).length, 0);
const summary = `<div class="summary">${
  temas.length ? `<div class="group"><span class="muted">Temas</span> ${TEMA_ORDER.filter((s) => count(temas, s)).map((s) => `${badge(s)} <strong>${count(temas, s)}</strong>`).join(" ")}</div>` : ""
}${pendingIdeas.length ? `<div class="group">${badge("idea", "Ideas por convertir")} <strong>${pendingIdeas.length}</strong></div>` : ""}${
  features.length
    ? `<div class="group"><span class="muted">Features</span> ${ORDER.filter((s) => count(features, s)).map((s) => `${badge(s)} <strong>${count(features, s)}</strong>`).join(" ")} <span class="muted">· tareas ${totals.done} de ${totals.total}</span></div>`
    : ""
}${openTotal ? `<div class="group">${badge("abierta")} <strong>${openTotal}</strong> <span class="muted">preguntas abiertas</span></div>` : ""}</div>`;

const featureRows = features
  .map(
    (f) =>
      `<tr><td>${numTag(f)}<a href="${esc(f.id)}.html">${esc(f.title)}</a><div class="muted note">${inline(f.summary)}</div></td><td>${esc(f.project)}</td><td>${badge(f.status)}</td><td class="nowrap">${openQuestions(f).length ? `<a href="${esc(f.id)}.html#preguntas-${esc(f.id)}">${qbadge(f)}</a>` : '<span class="muted">—</span>'}</td><td class="nowrap">${bar(f)}</td><td class="nowrap">${esc(f.updated)}</td></tr>`,
  )
  .join("\n");
const featuresTable = features.length
  ? `<table class="index"><thead><tr><th>Feature</th><th>Proyecto</th><th>Estado</th><th>Preguntas</th><th>Avance</th><th>Actualizado</th></tr></thead><tbody>
${featureRows}
</tbody></table>`
  : '<p class="muted">Ningún feature todavía.</p>';

const temaCard = (t) => {
  const pend = (t.ideas ?? []).filter((i) => (i.status ?? "idea") === "idea" || i.status === "propuesta").length;
  return `<li class="tema-card">${numTag(t)}<a href="${esc(t.id)}.html"><strong>${esc(t.title)}</strong></a><div class="meta">${badge(t.status ?? "por-explorar")} ${qbadge(t)} ${pend ? badge("idea", `${pend} ${pend === 1 ? "idea" : "ideas"}`) : ""}</div>${t.summary ? `<div class="muted note">${inline(t.summary)}</div>` : ""}<div class="row">${bar(t)}<span class="muted note">${esc(t.updated)}</span></div></li>`;
};
const learning = temas.filter((t) => !CLOSED_TEMA.has(t.status));
const areas = [...new Set(learning.map((t) => t.area || "Sin área"))].sort((a, b) => a.localeCompare(b, "es"));
const learningHtml = learning.length
  ? areas.map((a) => `<h3>${esc(a)}</h3>\n<ul class="tema-cards">\n${learning.filter((t) => (t.area || "Sin área") === a).map(temaCard).join("\n")}\n</ul>`).join("\n")
  : '<p class="muted">Ningún tema en estudio.</p>';
const closed = temas.filter((t) => CLOSED_TEMA.has(t.status));
const pendingIdeasHtml = pendingIdeas.length
  ? `<ul class="ideas">\n${pendingIdeas
      .sort((a, b) => IDEA_ORDER.indexOf(a.idea.status ?? "idea") - IDEA_ORDER.indexOf(b.idea.status ?? "idea"))
      .map((x) => ideaHtml(x.tema, x.idea).replace("</div>", ` <span class="muted note">de <a href="${esc(x.tema.id)}.html#idea-${esc(x.tema.id)}-${esc(x.idea.id)}">${esc(titled(x.tema))}</a></span></div>`))
      .join("\n")}\n</ul>`
  : '<p class="muted">No hay ideas por convertir.</p>';

writeFileSync(
  join(DIR, "index.html"),
  page(
    "Mesa de trabajo",
    `<h1>Mesa de trabajo</h1>
<p class="muted">Lo que se aprende y lo que se construye: los temas de estudio, las ideas que salen de ellos y los features en que se trabaja. <a href="todos.html">Ver todo en una página</a> · <a href="ideas.html">todas las ideas</a>.</p>
${summary}
<h2 id="aprendiendo">Aprendiendo</h2>
${learningHtml}
${closed.length ? `<p class="muted note">Dominados y archivados: ${closed.map((t) => `${numTag(t)}<a href="${esc(t.id)}.html">${esc(t.title)}</a> ${badge(t.status)}`).join(" · ")}</p>` : ""}
<h2 id="ideas">Ideas por convertir</h2>
<p class="muted note">Una idea se vuelve feature solo cuando el usuario lo decide.</p>
${pendingIdeasHtml}
<h2 id="features">Features</h2>
${featuresTable}`,
  ),
);

// ---------- Ideas por destino ----------

const targets = [...new Set(allIdeas.map((x) => ideaTarget(x.idea)))].sort((a, b) => (a === "" ? -1 : b === "" ? 1 : a.localeCompare(b, "es")));
writeFileSync(
  join(DIR, "ideas.html"),
  page(
    "Ideas",
    `<h1>Ideas</h1>
<p class="muted">Las ideas de los temas (proyectos nuevos y mejoras), agrupadas por el proyecto destino. Al convertir una se crea un feature en ese proyecto.</p>
${
  allIdeas.length
    ? targets
        .map(
          (tg) =>
            `<h2>${tg ? `<code>${esc(tg)}</code>` : "En esta carpeta"}</h2>\n<ul class="ideas">\n${allIdeas
              .filter((x) => ideaTarget(x.idea) === tg)
              .sort((a, b) => IDEA_ORDER.indexOf(a.idea.status ?? "idea") - IDEA_ORDER.indexOf(b.idea.status ?? "idea"))
              .map((x) => ideaHtml(x.tema, x.idea).replace("</div>", ` <span class="muted note">de <a href="${esc(x.tema.id)}.html#idea-${esc(x.tema.id)}-${esc(x.idea.id)}">${esc(titled(x.tema))}</a></span></div>`))
              .join("\n")}\n</ul>`,
        )
        .join("\n")
    : '<p class="muted">Ninguna idea todavía.</p>'
}`,
    `<span class="sep">›</span><span>Ideas</span>`,
  ),
);

// ---------- Acumulado ----------

const toc = (list) => list.map((f) => `<li>${numTag(f)}<a href="#${esc(f.id)}">${esc(f.title)}</a> ${badge(f.status)} ${qbadge(f)}</li>`).join("\n");
writeFileSync(
  join(DIR, "todos.html"),
  page(
    "Mesa de trabajo: acumulado",
    `<h1>Acumulado</h1>
${summary}
${temas.length ? `<h2>Temas</h2>\n<ul class="toc">\n${toc(temas)}\n</ul>` : ""}
${features.length ? `<h2>Features</h2>\n<ul class="toc">\n${toc(features)}\n</ul>` : ""}
${[...temas.map((t) => `<article class="tema">\n${sectionTema(t, 2, temaIds)}\n</article>`), ...features.map((f) => `<article>\n${sectionFeature(f, 2)}\n</article>`)].join("\n<hr>\n")}`,
    `<span class="sep">›</span><span>Acumulado</span>`,
  ),
);

console.log(
  `${features.length} features y ${temas.length} temas: index.html, ideas.html, todos.html${items.length ? " y " + items.map((f) => `${f.id}.html`).join(", ") : ""}`,
);
