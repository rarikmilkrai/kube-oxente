(function () {
  "use strict";

  const NS = "http://www.w3.org/2000/svg";
  const W = 210, H = 60;

  const COLORS = {
    internet: "#5aa9ff", lb: "#818cf8", cidr: "#8a93a6", ingress: "#a78bfa", service: "#38bdf8",
    workload: "#5fd38a", configmap: "#94a3b8", secret: "#f0a35e", pvc: "#c084fc", hpa: "#ffc04d",
    db: "#ff8f6b", cache: "#ff6b9d", queue: "#f7c948", storage: "#4fd1c5", search: "#63b3ed",
    aws: "#ff9900", external: "#9aa4b8", note: "#ffd166", user: "#e879f9", tf: "#8b5cf6",
    network: "#22d3ee", group: "#64748b", compute: "#34d399", security: "#f87171", namespace: "#5aa9ff"
  };

  const KIND_LABEL = {
    internet: "Internet", lb: "Load Balancer", cidr: "Bloco IP", ingress: "Ingress", service: "Service",
    workload: "Workload", configmap: "ConfigMap", secret: "Secret", pvc: "Volume (PVC)", hpa: "HPA",
    db: "Banco de dados", cache: "Cache", queue: "Fila / mensageria", storage: "Storage", search: "Busca",
    aws: "Serviço AWS", external: "Externo", note: "Nota", user: "Usuários", tf: "Recurso",
    network: "Rede", group: "Agrupador", compute: "Computação", security: "Segurança", namespace: "Namespace recolhido"
  };

  const ABBR = {
    internet: "WWW", lb: "LB", cidr: "IP", ingress: "ING", service: "SVC", configmap: "CM", secret: "SEC",
    pvc: "PVC", hpa: "HPA", db: "DB", cache: "RDS", queue: "MQ", storage: "S3", search: "OS", aws: "AWS",
    external: "EXT", note: "TXT", user: "USR", tf: "TF", network: "NET", group: "GRP", compute: "VM", security: "SG", namespace: "NS"
  };
  const WL_ABBR = { Deployment: "DEP", StatefulSet: "STS", DaemonSet: "DS", CronJob: "CJ" };

  const COLUMN = {
    namespace: 4, user: 0, internet: 0, cidr: 0, lb: 1, ingress: 2, service: 3, workload: 4, compute: 4,
    hpa: 5, configmap: 5, secret: 5, pvc: 5, note: 5, security: 2, network: 1, group: 1, tf: 4,
    db: 6, cache: 6, queue: 6, storage: 6, search: 6, aws: 6, external: 6
  };

  const EDGE = {
    traffic: { c: "#5aa9ff", w: 2, dash: "6 4", flow: true },
    routes: { c: "#5aa9ff", w: 1.6 },
    calls: { c: "#5fd38a", w: 1.8, dash: "5 5", flow: true },
    config: { c: "#5b6478", w: 1.1, dash: "2 4" },
    storage: { c: "#c084fc", w: 1.3 },
    scales: { c: "#ffc04d", w: 1.3, dash: "6 4" },
    netpol: { c: "#fb923c", w: 1.3, dash: "3 3" },
    dns: { c: "#8a93a6", w: 1.2, dash: "4 3" },
    ref: { c: "#8b5cf6", w: 1.5 },
    depends: { c: "#a78bfa", w: 1.3, dash: "5 4" },
    custom: { c: "#e879f9", w: 1.8, dash: "7 5", flow: true }
  };

  const LEGEND = [
    ["traffic", "tráfego"], ["calls", "chama (env/config)"], ["config", "config/secret"],
    ["storage", "volume"], ["scales", "escala"], ["netpol", "NetworkPolicy"], ["custom", "manual"]
  ];

  const LAYERS = [
    { id: "config", label: "ConfigMaps/Secrets", c: COLORS.configmap, kinds: ["configmap", "secret"] },
    { id: "storage", label: "Volumes", c: COLORS.pvc, kinds: ["pvc"] },
    { id: "hpa", label: "HPA", c: COLORS.hpa, kinds: ["hpa"] },
    { id: "external", label: "Externos", c: COLORS.db, kinds: ["db", "cache", "queue", "storage", "search", "aws", "external"] },
    { id: "netpol", label: "NetworkPolicy", c: "#fb923c", kinds: ["cidr"] },
    { id: "labels", label: "Rótulos", c: "#e7e9ef", kinds: [] },
    { id: "system", label: "Sistema", c: "#7a8194", kinds: [] }
  ];

  const ENV_C = { prod: "#ff6b6b", staging: "#ffc04d", dev: "#5aa9ff", other: "#7a8194" };

  function envOf(s) {
    if (/(prod|prd|production|live)/i.test(s)) return "prod";
    if (/(stg|staging|homolog|hml|hmg|uat|qa)/i.test(s)) return "staging";
    if (/(dev|develop|sandbox|sbx|test)/i.test(s)) return "dev";
    return "other";
  }

  function h(s) { return (s == null ? "" : "" + s).replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;" }[c])); }

  function el(tag, attrs, parent) {
    const e = document.createElementNS(NS, tag);
    if (attrs) for (const k in attrs) if (attrs[k] != null) e.setAttribute(k, attrs[k]);
    if (parent) parent.appendChild(e);
    return e;
  }

  function trunc(s, n) { s = s || ""; return s.length > n ? s.slice(0, n - 1) + "…" : s; }

  function lsGet(k, def) {
    try { const v = localStorage.getItem(k); return v ? JSON.parse(v) : def; } catch (e) { return def; }
  }
  function lsSet(k, v) { try { localStorage.setItem(k, JSON.stringify(v)); } catch (e) { } }

  function download(name, blob) {
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = name;
    document.body.appendChild(a);
    a.click();
    setTimeout(() => { URL.revokeObjectURL(a.href); a.remove(); }, 500);
  }

  function computeLayout(nodes, edges) {
    const pos = {};
    const gapX = 90, rowH = H + 26, bandPad = 46;
    const colX = c => c * (W + gapX);
    const byId = {};
    nodes.forEach(n => { byId[n.id] = n; });
    const adj = {};
    nodes.forEach(n => { adj[n.id] = []; });
    edges.forEach(e => { if (adj[e.from] && adj[e.to]) { adj[e.from].push(e.to); adj[e.to].push(e.from); } });

    const groups = {}, globals = [];
    nodes.forEach(n => {
      if (n.group) (groups[n.group] = groups[n.group] || []).push(n);
      else globals.push(n);
    });
    const gnames = Object.keys(groups).sort();
    const rank = {};
    let y = 0;
    gnames.forEach(g => {
      const cols = {};
      groups[g].forEach(n => { const c = COLUMN[n.kind] != null ? COLUMN[n.kind] : 4; (cols[c] = cols[c] || []).push(n); });
      const order = [4, 3, 2, 5, 1, 0, 6];
      order.forEach(c => {
        const list = cols[c];
        if (!list) return;
        if (c === 4) list.sort((a, b) => a.name.localeCompare(b.name));
        else {
          const bary = n => {
            const r = adj[n.id].map(x => rank[x]).filter(v => v != null);
            return r.length ? r.reduce((a, b) => a + b, 0) / r.length : 1e9;
          };
          list.sort((a, b) => bary(a) - bary(b) || a.name.localeCompare(b.name));
        }
        list.forEach((n, i) => { rank[n.id] = i; });
      });
      let maxRows = 1;
      Object.keys(cols).forEach(c => { maxRows = Math.max(maxRows, cols[c].length); });
      Object.keys(cols).forEach(c => {
        const list = cols[c];
        const off = (maxRows - list.length) * rowH / 2;
        list.forEach((n, i) => { pos[n.id] = { x: colX(+c), y: y + bandPad + off + i * rowH }; rank[n.id] = (pos[n.id].y); });
      });
      y += bandPad + maxRows * rowH + 40;
    });

    const gcols = {};
    globals.forEach(n => { const c = COLUMN[n.kind] != null ? COLUMN[n.kind] : 6; (gcols[c] = gcols[c] || []).push(n); });
    const totalH = Math.max(y, rowH * 3);
    [1, 0, 6, 2, 3, 4, 5].forEach(c => {
      const list = gcols[c];
      if (!list) return;
      const bary = n => {
        const r = adj[n.id].map(x => pos[x] && pos[x].y).filter(v => v != null);
        return r.length ? r.reduce((a, b) => a + b, 0) / r.length : totalH / 2;
      };
      list.forEach(n => { n._b = bary(n); });
      list.sort((a, b) => a._b - b._b);
      let last = -1e9;
      const x = c === 6 ? colX(6) : (c <= 1 ? colX(c) - (c === 0 && list.length ? 0 : 0) : colX(c));
      list.forEach(n => {
        let yy = Math.max(n._b - H / 2, last + rowH);
        pos[n.id] = { x: x, y: yy };
        last = yy;
        delete n._b;
      });
    });
    return pos;
  }

  function createDiagram(stage, opts) {
    opts = opts || {};
    const svg = el("svg", { xmlns: NS });
    stage.appendChild(svg);
    const defs = el("defs", null, svg);
    const pat = el("pattern", { id: "ar-grid-" + opts.key, width: 22, height: 22, patternUnits: "userSpaceOnUse" }, defs);
    el("circle", { cx: 1, cy: 1, r: 1, fill: "#1f2433" }, pat);
    const glow = el("filter", { id: "ar-glow-" + opts.key, x: "-20%", y: "-40%", width: "140%", height: "180%" }, defs);
    el("feDropShadow", { dx: 0, dy: 0, stdDeviation: 5, "flood-color": "#a78bfa", "flood-opacity": .55 }, glow);
    const shadow = el("filter", { id: "ar-sh-" + opts.key, x: "-10%", y: "-20%", width: "120%", height: "160%" }, defs);
    el("feDropShadow", { dx: 0, dy: 3, stdDeviation: 4, "flood-color": "#000", "flood-opacity": .45 }, shadow);
    Object.keys(EDGE).forEach(k => {
      const m = el("marker", { id: "ar-m-" + opts.key + "-" + k, viewBox: "0 0 10 10", refX: 9, refY: 5, markerWidth: 7, markerHeight: 7, orient: "auto-start-reverse" }, defs);
      el("path", { d: "M0,1 L9,5 L0,9 z", fill: EDGE[k].c }, m);
    });
    const bg = el("rect", { x: -50000, y: -50000, width: 100000, height: 100000, fill: "url(#ar-grid-" + opts.key + ")" }, svg);
    const vp = el("g", null, svg);
    bg.remove();
    vp.appendChild(bg);
    const gG = el("g", null, vp), gE = el("g", null, vp), gL = el("g", null, vp), gN = el("g", null, vp), gT = el("g", null, vp);

    const st = {
      nodes: [], edges: [], byId: {}, pos: {}, view: { x: 40, y: 40, k: 1 }, sel: null, hover: null,
      nodeEl: {}, edgeEl: {}, labelEl: {}, groupEl: {}, connect: false, connectFrom: null, showLabels: false, match: null
    };

    function applyView() {
      vp.setAttribute("transform", "translate(" + st.view.x + "," + st.view.y + ") scale(" + st.view.k + ")");
      if (opts.onZoom) opts.onZoom(st.view.k);
    }

    function toWorld(cx, cy) {
      const r = svg.getBoundingClientRect();
      return { x: (cx - r.left - st.view.x) / st.view.k, y: (cy - r.top - st.view.y) / st.view.k };
    }

    function ports(a, b) {
      const pa = st.pos[a], pb = st.pos[b];
      if (!pa || !pb) return null;
      const acx = pa.x + W / 2, bcx = pb.x + W / 2;
      if (pb.x >= pa.x + W * 0.6) return [pa.x + W, pa.y + H / 2, pb.x, pb.y + H / 2, "h"];
      if (pb.x + W * 0.6 <= pa.x) return [pa.x, pa.y + H / 2, pb.x + W, pb.y + H / 2, "h"];
      if (pb.y > pa.y) return [acx, pa.y + H, bcx, pb.y, "v"];
      return [acx, pa.y, bcx, pb.y + H, "v"];
    }

    function edgePath(e) {
      const p = ports(e.from, e.to);
      if (!p) return null;
      const [sx, sy, tx, ty, o] = p;
      let d, mx, my;
      if (o === "h") {
        const dx = Math.max(40, Math.abs(tx - sx) * 0.5) * (tx >= sx ? 1 : -1);
        d = "M" + sx + "," + sy + " C" + (sx + dx) + "," + sy + " " + (tx - dx) + "," + ty + " " + tx + "," + ty;
      } else {
        const dy = Math.max(30, Math.abs(ty - sy) * 0.5) * (ty >= sy ? 1 : -1);
        d = "M" + sx + "," + sy + " C" + sx + "," + (sy + dy) + " " + tx + "," + (ty - dy) + " " + tx + "," + ty;
      }
      mx = (sx + tx) / 2; my = (sy + ty) / 2;
      return { d: d, mx: mx, my: my };
    }

    function drawEdge(e) {
      const s = EDGE[e.kind] || EDGE.ref;
      const p = edgePath(e);
      if (!p) return;
      const g = el("g", { "data-eid": e.id, class: "ar-e", "data-w": s.w }, gE);
      el("path", { d: p.d, stroke: "transparent", "stroke-width": 12, fill: "none" }, g);
      const path = el("path", {
        d: p.d, class: "ar-edge" + (s.flow ? " ar-flow" : ""), stroke: s.c, "stroke-width": s.w,
        "stroke-dasharray": s.dash || null, "stroke-opacity": .85, "marker-end": "url(#ar-m-" + opts.key + "-" + (EDGE[e.kind] ? e.kind : "ref") + ")"
      }, g);
      const t = el("title", null, g);
      t.textContent = (e.label ? e.label + "\n" : "") + (e.evidence || []).join("\n");
      st.edgeEl[e.id] = g;
      if (e.label) {
        const lg = el("g", { "data-eid": e.id, display: st.showLabels ? null : "none" }, gL);
        const txt = trunc(e.label, 34);
        const tw = txt.length * 5.6 + 10;
        el("rect", { x: p.mx - tw / 2, y: p.my - 9, width: tw, height: 16, rx: 5, fill: "#0e1016", stroke: s.c, "stroke-opacity": .5 }, lg);
        const tt = el("text", { x: p.mx, y: p.my + 3, "text-anchor": "middle", "font-size": 10, fill: "#cbd5e1" }, lg);
        tt.textContent = txt;
        st.labelEl[e.id] = lg;
      }
      return path;
    }

    function updateEdge(e) {
      const g = st.edgeEl[e.id];
      if (!g) return;
      const p = edgePath(e);
      if (!p) return;
      g.childNodes[0].setAttribute("d", p.d);
      g.childNodes[1].setAttribute("d", p.d);
      const lg = st.labelEl[e.id];
      if (lg) {
        const r = lg.childNodes[0], t = lg.childNodes[1];
        const tw = +r.getAttribute("width");
        r.setAttribute("x", p.mx - tw / 2); r.setAttribute("y", p.my - 9);
        t.setAttribute("x", p.mx); t.setAttribute("y", p.my + 3);
      }
    }

    function drawNode(n) {
      const p = st.pos[n.id];
      const c = n.color || COLORS[n.kind] || COLORS.tf;
      const g = el("g", { "data-nid": n.id, class: "ar-n", transform: "translate(" + p.x + "," + p.y + ")" }, gN);
      if (n.kind === "note") {
        el("rect", { width: W, height: H, rx: 6, fill: "#3a3420", stroke: c, "stroke-opacity": .7, filter: "url(#ar-sh-" + opts.key + ")" }, g);
        const t = el("text", { x: 12, y: 20, "font-size": 12, fill: "#fde68a", "font-weight": 600 }, g);
        t.textContent = trunc(n.name, 30);
        const t2 = el("text", { x: 12, y: 37, "font-size": 10.5, fill: "#d6c58a" }, g);
        t2.textContent = trunc(n.sub, 36);
      } else {
        const sc = n.sev === 2 ? "#ff6b6b" : c;
        el("rect", {
          width: W, height: H, rx: 11, fill: "#151924", stroke: sc, "stroke-opacity": n.sev ? 1 : .5,
          "stroke-width": n.sev ? 1.6 : 1.1, "stroke-dasharray": n.custom ? "5 3" : (n.kind === "namespace" ? "6 3" : null),
          filter: "url(#ar-sh-" + opts.key + ")", class: n.sev === 2 ? "ar-pulse" : null
        }, g);
        if (n.sev === 2) el("rect", { width: W, height: H, rx: 11, fill: "#ff6b6b", "fill-opacity": .08 }, g);
        el("rect", { x: 9, y: 13, width: 34, height: 34, rx: 8, fill: c, "fill-opacity": .16, stroke: c, "stroke-opacity": .35 }, g);
        const ab = n.abbr || (n.kind === "workload" && n.meta && WL_ABBR[n.meta.kind]) || ABBR[n.kind] || "•";
        const at = el("text", { x: 26, y: 34, "text-anchor": "middle", "font-size": ab.length > 3 ? 8.5 : 10, "font-weight": 700, fill: c, "letter-spacing": .3 }, g);
        at.textContent = ab;
        const m = n.meta || {};
        const hasM = m.pods != null;
        const nt = el("text", { x: 52, y: hasM ? 20 : 25, "font-size": 12.5, "font-weight": 600, fill: "#e7e9ef" }, g);
        nt.textContent = trunc(n.name, 22);
        const sb = el("text", { x: 52, y: hasM ? 35 : 42, "font-size": 10.5, fill: "#8a93a6" }, g);
        sb.textContent = trunc(n.sub || KIND_LABEL[n.kind] || "", 29);
        if (hasM) {
          const mt = el("text", { x: 52, y: 50, "font-size": 10, fill: "#6b7488", "font-family": "ui-monospace,Menlo,Consolas,monospace" }, g);
          const span = (txt, color) => { const t = el("tspan", { fill: color || null }, mt); t.textContent = txt; };
          const ready = +m.podsReady, total = +m.pods;
          span(ready + "/" + total, total && ready < total ? (ready ? "#ffc04d" : "#ff6b6b") : "#5fd38a");
          if (m.cpu) span("  " + m.cpu);
          if (m.mem) span("  " + m.mem);
          if (+m.restarts) span("  ↻" + m.restarts, "#ffc04d");
          if (m.reason && n.sev) span("  " + trunc(m.reason, 14), n.sev === 2 ? "#ff6b6b" : "#ffc04d");
        }
      }
      if (n.sev) {
        const sc = n.sev === 2 ? "#ff6b6b" : "#ffc04d";
        el("circle", { cx: W - 4, cy: 4, r: 8, fill: sc, stroke: "#0b0d13", "stroke-width": 2 }, g);
        const ex = el("text", { x: W - 4, y: 8, "text-anchor": "middle", "font-size": 11, "font-weight": 800, fill: "#0b0d13" }, g);
        ex.textContent = "!";
      }
      el("circle", { cx: W, cy: H / 2, r: 5.5, fill: "#a78bfa", stroke: "#0b0d13", "stroke-width": 2, class: "ar-handle", "data-handle": n.id }, g);
      const tt = el("title", null, g);
      tt.textContent = n.name + (n.ns ? "\nnamespace: " + n.ns : "") + (n.sub ? "\n" + n.sub : "");
      st.nodeEl[n.id] = g;
    }

    function groupBox(gid) {
      let x1 = 1e9, y1 = 1e9, x2 = -1e9, y2 = -1e9, any = false;
      st.nodes.forEach(n => {
        if (n.group !== gid) return;
        const p = st.pos[n.id];
        if (!p) return;
        any = true;
        x1 = Math.min(x1, p.x); y1 = Math.min(y1, p.y); x2 = Math.max(x2, p.x + W); y2 = Math.max(y2, p.y + H);
      });
      if (!any) return null;
      return { x: x1 - 20, y: y1 - 38, w: x2 - x1 + 40, h: y2 - y1 + 58 };
    }

    function drawGroups() {
      gG.innerHTML = "";
      st.groupEl = {};
      const gids = [];
      st.nodes.forEach(n => { if (n.group && gids.indexOf(n.group) < 0) gids.push(n.group); });
      gids.forEach(gid => {
        const b = groupBox(gid);
        if (!b) return;
        const label = (opts.groupLabel && opts.groupLabel(gid)) || gid;
        const c = (opts.groupColor && opts.groupColor(gid)) || ENV_C[envOf(label)];
        const g = el("g", { "data-gid": gid }, gG);
        el("rect", { x: b.x, y: b.y, width: b.w, height: b.h, rx: 16, fill: c, "fill-opacity": .035, stroke: c, "stroke-opacity": .35, "stroke-dasharray": "6 5" }, g);
        const lg = el("g", { class: "ar-glabel", "data-glabel": gid }, g);
        const tw = Math.min(label.length * 7 + 46, b.w - 10);
        el("rect", { x: b.x + 12, y: b.y + 9, width: tw, height: 20, rx: 6, fill: c, "fill-opacity": .14 }, lg);
        const t = el("text", { x: b.x + 22, y: b.y + 23, "font-size": 11.5, "font-weight": 700, fill: c }, lg);
        t.textContent = trunc(label, Math.max(6, Math.floor((tw - 46) / 7) + 4));
        const tag = opts.groupTag ? opts.groupTag(gid) : "namespace";
        const t2 = el("text", { x: b.x + 12 + tw + 8, y: b.y + 23, "font-size": 10, fill: "#5b6478" }, lg);
        t2.textContent = tag;
        if (opts.onGroupCollapse) {
          const cb = el("g", { "data-gcollapse": gid, style: "cursor:pointer" }, g);
          el("rect", { x: b.x + b.w - 34, y: b.y + 9, width: 22, height: 20, rx: 6, fill: c, "fill-opacity": .14 }, cb);
          const ct = el("text", { x: b.x + b.w - 23, y: b.y + 24, "text-anchor": "middle", "font-size": 14, "font-weight": 700, fill: c }, cb);
          ct.textContent = "−";
          const tt = el("title", null, cb);
          tt.textContent = "recolher namespace";
        }
        st.groupEl[gid] = g;
      });
    }

    function renderAll() {
      gE.innerHTML = ""; gL.innerHTML = ""; gN.innerHTML = "";
      st.nodeEl = {}; st.edgeEl = {}; st.labelEl = {};
      drawGroups();
      st.edges.forEach(drawEdge);
      st.nodes.forEach(drawNode);
      applyHighlight();
    }

    function neighborSet(id) {
      const s = new Set([id]);
      const es = new Set();
      st.edges.forEach(e => { if (e.from === id || e.to === id) { s.add(e.from); s.add(e.to); es.add(e.id); } });
      return { nodes: s, edges: es };
    }

    function applyHighlight() {
      const focus = st.hover || (st.sel && st.sel.type === "node" ? st.sel.id : null);
      let nb = null;
      if (focus && st.byId[focus]) nb = neighborSet(focus);
      if (st.sel && st.sel.type === "edge") {
        const e = st.edges.find(x => x.id === st.sel.id);
        if (e) nb = { nodes: new Set([e.from, e.to]), edges: new Set([e.id]) };
      }
      if (st.match && st.match.size) nb = { nodes: st.match, edges: new Set() };
      if (st.impact) nb = st.impact;
      Object.keys(st.nodeEl).forEach(id => {
        const g = st.nodeEl[id];
        g.classList.toggle("ar-dim", !!nb && !nb.nodes.has(id));
        const isSel = st.sel && st.sel.type === "node" && st.sel.id === id;
        g.classList.toggle("sel", !!isSel);
        g.firstChild.setAttribute("filter", isSel || (st.connectFrom === id) ? "url(#ar-glow-" + opts.key + ")" : "url(#ar-sh-" + opts.key + ")");
        g.classList.toggle("ar-imp", !!st.impact && st.impact.nodes.has(id) && st.impact.root !== id);
        g.classList.toggle("ar-imp-root", !!st.impact && st.impact.root === id);
      });
      Object.keys(st.edgeEl).forEach(id => {
        const on = !nb || nb.edges.has(id);
        st.edgeEl[id].classList.toggle("ar-dim", !on);
        st.edgeEl[id].classList.toggle("ar-imp-e", !!st.impact && st.impact.edges.has(id));
        const lg = st.labelEl[id];
        if (lg) lg.setAttribute("display", (st.showLabels && on) || (nb && nb.edges.has(id)) ? "inline" : "none");
        const w = +st.edgeEl[id].getAttribute("data-w");
        st.edgeEl[id].childNodes[1].setAttribute("stroke-width", nb && nb.edges.has(id) ? w + 1 : w);
      });
    }

    function moveNode(id, x, y) {
      st.pos[id] = { x: x, y: y };
      const g = st.nodeEl[id];
      if (g) g.setAttribute("transform", "translate(" + x + "," + y + ")");
      st.edges.forEach(e => { if (e.from === id || e.to === id) updateEdge(e); });
    }

    let drag = null;
    svg.addEventListener("pointerdown", ev => {
      if (ev.button !== 0) return;
      const t = ev.target;
      const gc = t.closest("[data-gcollapse]");
      if (gc) { if (opts.onGroupCollapse) opts.onGroupCollapse(gc.getAttribute("data-gcollapse")); return; }
      const handle = t.closest("[data-handle]");
      const nodeG = t.closest("[data-nid]");
      const glabel = t.closest("[data-glabel]");
      const edgeG = t.closest("[data-eid]");
      const w = toWorld(ev.clientX, ev.clientY);
      svg.setPointerCapture(ev.pointerId);
      if (handle) {
        const id = handle.getAttribute("data-handle");
        const p = st.pos[id];
        const line = el("path", { d: "", stroke: "#e879f9", "stroke-width": 2, "stroke-dasharray": "6 4", fill: "none" }, gT);
        drag = { type: "link", from: id, line: line, sx: p.x + W, sy: p.y + H / 2 };
        return;
      }
      if (nodeG) {
        const id = nodeG.getAttribute("data-nid");
        const p = st.pos[id];
        drag = { type: "node", id: id, dx: w.x - p.x, dy: w.y - p.y, moved: false, x0: ev.clientX, y0: ev.clientY };
        return;
      }
      if (glabel) {
        const gid = glabel.getAttribute("data-glabel");
        const members = st.nodes.filter(n => n.group === gid).map(n => ({ id: n.id, x: st.pos[n.id].x, y: st.pos[n.id].y }));
        drag = { type: "group", gid: gid, sx: w.x, sy: w.y, members: members, moved: false };
        return;
      }
      if (edgeG) {
        drag = { type: "edgeclick", id: edgeG.getAttribute("data-eid") };
        return;
      }
      drag = { type: "pan", x0: ev.clientX, y0: ev.clientY, vx: st.view.x, vy: st.view.y, moved: false };
      svg.classList.add("panning");
    });

    svg.addEventListener("pointermove", ev => {
      if (!drag) {
        const nodeG = ev.target.closest && ev.target.closest("[data-nid]");
        const id = nodeG ? nodeG.getAttribute("data-nid") : null;
        if (id !== st.hover) { st.hover = id; applyHighlight(); }
        return;
      }
      const w = toWorld(ev.clientX, ev.clientY);
      if (drag.type === "node") {
        if (!drag.moved && Math.abs(ev.clientX - drag.x0) + Math.abs(ev.clientY - drag.y0) < 4) return;
        drag.moved = true;
        moveNode(drag.id, Math.round(w.x - drag.dx), Math.round(w.y - drag.dy));
        const n = st.byId[drag.id];
        if (n && n.group) drawGroups();
      } else if (drag.type === "group") {
        drag.moved = true;
        const dx = w.x - drag.sx, dy = w.y - drag.sy;
        drag.members.forEach(m => moveNode(m.id, Math.round(m.x + dx), Math.round(m.y + dy)));
        drawGroups();
      } else if (drag.type === "pan") {
        drag.moved = drag.moved || Math.abs(ev.clientX - drag.x0) + Math.abs(ev.clientY - drag.y0) > 3;
        st.view.x = drag.vx + ev.clientX - drag.x0;
        st.view.y = drag.vy + ev.clientY - drag.y0;
        applyView();
      } else if (drag.type === "link") {
        const dx = Math.max(40, Math.abs(w.x - drag.sx) * .5);
        drag.line.setAttribute("d", "M" + drag.sx + "," + drag.sy + " C" + (drag.sx + dx) + "," + drag.sy + " " + (w.x - dx) + "," + w.y + " " + w.x + "," + w.y);
      }
    });

    svg.addEventListener("pointerup", ev => {
      const d = drag;
      drag = null;
      svg.classList.remove("panning");
      if (!d) return;
      if (d.type === "node") {
        if (d.moved) { if (opts.onMove) opts.onMove(); }
        else if (st.connect) {
          if (!st.connectFrom) { st.connectFrom = d.id; applyHighlight(); if (opts.onConnectStart) opts.onConnectStart(d.id); }
          else { const from = st.connectFrom; st.connectFrom = null; applyHighlight(); if (from !== d.id && opts.onConnect) opts.onConnect(from, d.id); }
        } else select({ type: "node", id: d.id });
      } else if (d.type === "group") {
        if (d.moved && opts.onMove) opts.onMove();
      } else if (d.type === "edgeclick") {
        select({ type: "edge", id: d.id });
      } else if (d.type === "pan") {
        if (opts.onView) opts.onView();
        if (!d.moved) select(null);
      } else if (d.type === "link") {
        d.line.remove();
        const t = document.elementFromPoint(ev.clientX, ev.clientY);
        const ng = t && t.closest && t.closest("[data-nid]");
        if (ng) {
          const to = ng.getAttribute("data-nid");
          if (to !== d.from && opts.onConnect) opts.onConnect(d.from, to);
        }
      }
    });

    svg.addEventListener("pointerleave", () => { if (!drag && st.hover) { st.hover = null; applyHighlight(); } });

    svg.addEventListener("dblclick", ev => {
      const nodeG = ev.target.closest("[data-nid]");
      if (nodeG) { if (opts.onDblNode) opts.onDblNode(nodeG.getAttribute("data-nid")); return; }
      if (ev.target.closest("[data-eid]") || ev.target.closest("[data-glabel]")) return;
      const w = toWorld(ev.clientX, ev.clientY);
      const r = stage.getBoundingClientRect();
      if (opts.onDblEmpty) opts.onDblEmpty(w, ev.clientX - r.left, ev.clientY - r.top);
    });

    svg.addEventListener("wheel", ev => {
      ev.preventDefault();
      const r = svg.getBoundingClientRect();
      zoomAt(Math.exp(-ev.deltaY * 0.0015), ev.clientX - r.left, ev.clientY - r.top);
    }, { passive: false });

    let viewTimer = null;
    function zoomAt(f, cx, cy) {
      const k = Math.min(2.5, Math.max(0.08, st.view.k * f));
      const wx = (cx - st.view.x) / st.view.k, wy = (cy - st.view.y) / st.view.k;
      st.view.k = k;
      st.view.x = cx - wx * k;
      st.view.y = cy - wy * k;
      applyView();
      clearTimeout(viewTimer);
      viewTimer = setTimeout(() => { if (opts.onView) opts.onView(); }, 300);
    }

    function select(s) {
      st.sel = s;
      applyHighlight();
      if (opts.onSelect) opts.onSelect(s);
    }

    function bbox() {
      let x1 = 1e9, y1 = 1e9, x2 = -1e9, y2 = -1e9;
      st.nodes.forEach(n => {
        const p = st.pos[n.id];
        if (!p) return;
        x1 = Math.min(x1, p.x); y1 = Math.min(y1, p.y - (n.group ? 40 : 0)); x2 = Math.max(x2, p.x + W); y2 = Math.max(y2, p.y + H);
      });
      if (x1 > x2) return { x: 0, y: 0, w: 400, h: 300 };
      return { x: x1 - 30, y: y1 - 30, w: x2 - x1 + 60, h: y2 - y1 + 60 };
    }

    function fit() {
      const b = bbox();
      const r = svg.getBoundingClientRect();
      if (!r.width) return;
      const k = Math.min(1.15, Math.max(0.08, Math.min(r.width / b.w, r.height / b.h) * 0.95));
      st.view.k = k;
      st.view.x = (r.width - b.w * k) / 2 - b.x * k;
      st.view.y = (r.height - b.h * k) / 2 - b.y * k;
      applyView();
      if (opts.onView) opts.onView();
    }

    function center(id) {
      const p = st.pos[id];
      if (!p) return;
      const r = svg.getBoundingClientRect();
      const k = Math.max(st.view.k, 0.8);
      st.view.k = k;
      st.view.x = r.width / 2 - (p.x + W / 2) * k;
      st.view.y = r.height / 2 - (p.y + H / 2) * k;
      applyView();
    }

    function setData(nodes, edges, storedPos, relayout) {
      st.nodes = nodes;
      st.byId = {};
      nodes.forEach(n => { st.byId[n.id] = n; });
      st.edges = edges.filter(e => st.byId[e.from] && st.byId[e.to]);
      const lay = computeLayout(nodes, st.edges);
      const pos = {};
      nodes.forEach(n => {
        const sp = !relayout && storedPos && storedPos[n.id];
        pos[n.id] = sp ? { x: sp.x, y: sp.y } : (st.pos[n.id] && !relayout ? st.pos[n.id] : lay[n.id]);
      });
      st.pos = pos;
      if (st.sel && st.sel.type === "node" && !st.byId[st.sel.id]) st.sel = null;
      renderAll();
    }

    function exportSVGString() {
      const b = bbox();
      const clone = vp.cloneNode(true);
      clone.removeAttribute("transform");
      clone.querySelectorAll(".ar-handle").forEach(x => x.remove());
      clone.querySelectorAll("title").forEach(x => x.remove());
      clone.querySelectorAll(".ar-dim").forEach(x => x.classList.remove("ar-dim"));
      clone.firstChild.remove();
      const defsC = defs.cloneNode(true);
      const head = '<svg xmlns="http://www.w3.org/2000/svg" width="' + Math.round(b.w) + '" height="' + Math.round(b.h) +
        '" viewBox="' + b.x + " " + b.y + " " + b.w + " " + b.h + '" font-family="Inter,ui-sans-serif,system-ui,Segoe UI,Roboto,sans-serif">';
      const bgR = '<rect x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" fill="#0b0d13"/>';
      return head + new XMLSerializer().serializeToString(defsC) + bgR + new XMLSerializer().serializeToString(clone) + "</svg>";
    }

    function exportSVG(name) {
      download(name + ".svg", new Blob([exportSVGString()], { type: "image/svg+xml" }));
    }

    function exportPNG(name) {
      const s = exportSVGString();
      const b = bbox();
      const scale = Math.min(2, 12000 / Math.max(b.w, b.h));
      const img = new Image();
      img.onload = () => {
        const c = document.createElement("canvas");
        c.width = Math.round(b.w * scale); c.height = Math.round(b.h * scale);
        const ctx = c.getContext("2d");
        ctx.scale(scale, scale);
        ctx.drawImage(img, 0, 0, b.w, b.h);
        c.toBlob(bl => download(name + ".png", bl), "image/png");
      };
      img.src = "data:image/svg+xml;charset=utf-8," + encodeURIComponent(s);
    }

    const DEP_KINDS = new Set(["traffic", "routes", "calls", "config", "storage", "dns", "custom", "ref", "depends"]);
    function impactOf(root) {
      const nodes = new Set([root]), edges = new Set();
      const queue = [root];
      while (queue.length) {
        const cur = queue.shift();
        st.edges.forEach(e => {
          if (e.to !== cur || !DEP_KINDS.has(e.kind)) return;
          edges.add(e.id);
          if (!nodes.has(e.from)) { nodes.add(e.from); queue.push(e.from); }
        });
      }
      return { root: root, nodes: nodes, edges: edges };
    }

    function groupsOf() {
      const out = {};
      st.nodes.forEach(n => { if (n.group && !out[n.group]) { const b = groupBox(n.group); if (b) out[n.group] = b; } });
      return out;
    }

    function mmdText(s) { return (s || "").replace(/"/g, "#quot;").replace(/[<>]/g, ""); }

    function exportMermaid() {
      const ids = {};
      st.nodes.forEach((n, i) => { ids[n.id] = "n" + i; });
      const shape = (n, label) => {
        const l = '"' + label + '"';
        switch (n.kind) {
          case "db": case "pvc": case "storage": return "[(" + l + ")]";
          case "service": return "([" + l + "])";
          case "ingress": return ">" + l + "]";
          case "internet": case "user": return "((" + l + "))";
          case "lb": case "external": case "aws": case "cidr": return "{{" + l + "}}";
          case "configmap": case "secret": return "[/" + l + "/]";
          case "queue": case "cache": return "[[" + l + "]]";
          case "namespace": return "[" + l + "]";
          default: return "[" + l + "]";
        }
      };
      const line = n => ids[n.id] + shape(n, mmdText(n.name) + (n.sub ? "<br/><small>" + mmdText(n.sub) + "</small>" : ""));
      const out = ["flowchart LR"];
      const kinds = new Set(st.nodes.map(n => n.kind));
      kinds.forEach(k => out.push("  classDef " + k + " fill:" + (COLORS[k] || COLORS.tf) + "22,stroke:" + (COLORS[k] || COLORS.tf) + ",stroke-width:1.5px"));
      const groups = {};
      st.nodes.forEach(n => { if (n.group) (groups[n.group] = groups[n.group] || []).push(n); });
      Object.keys(groups).sort().forEach((gid, i) => {
        const label = (opts.groupLabel && opts.groupLabel(gid)) || gid;
        out.push("  subgraph g" + i + '["' + mmdText(label) + '"]');
        groups[gid].forEach(n => out.push("    " + line(n)));
        out.push("  end");
      });
      st.nodes.forEach(n => { if (!n.group) out.push("  " + line(n)); });
      const styles = [];
      st.edges.forEach((e, i) => {
        const s = EDGE[e.kind] || EDGE.ref;
        const arrow = e.kind === "traffic" ? "==>" : (s.dash ? "-.->" : "-->");
        const lbl = e.label ? '|"' + mmdText(trunc(e.label, 40)) + '"|' : "";
        out.push("  " + ids[e.from] + " " + arrow + lbl + " " + ids[e.to]);
        styles.push("  linkStyle " + i + " stroke:" + s.c);
      });
      st.nodes.forEach(n => out.push("  class " + ids[n.id] + " " + n.kind));
      return out.concat(styles).join("\n") + "\n";
    }

    function xmlEsc(s) { return (s == null ? "" : "" + s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;"); }

    function exportDrawio() {
      const cells = ['<mxCell id="0"/>', '<mxCell id="1" parent="0"/>'];
      const groups = groupsOf();
      const gids = {};
      Object.keys(groups).sort().forEach((gid, i) => {
        const b = groups[gid];
        const label = (opts.groupLabel && opts.groupLabel(gid)) || gid;
        const c = (opts.groupColor && opts.groupColor(gid)) || ENV_C[envOf(label)];
        gids[gid] = "g" + i;
        cells.push('<mxCell id="g' + i + '" value="' + xmlEsc(label) + '" style="swimlane;rounded=1;arcSize=4;dashed=1;fillColor=none;swimlaneFillColor=none;strokeColor=' + c +
          ";fontColor=" + c + ';fontStyle=1;startSize=26;align=left;spacingLeft=10;" vertex="1" parent="1"><mxGeometry x="' + Math.round(b.x) + '" y="' + Math.round(b.y) +
          '" width="' + Math.round(b.w) + '" height="' + Math.round(b.h) + '" as="geometry"/></mxCell>');
      });
      const ids = {};
      st.nodes.forEach((n, i) => {
        ids[n.id] = "n" + i;
        const p = st.pos[n.id];
        const c = COLORS[n.kind] || COLORS.tf;
        let shape = "rounded=1;arcSize=14;";
        if (n.kind === "db" || n.kind === "pvc" || n.kind === "storage") shape = "shape=cylinder3;boundedLbl=1;size=8;";
        else if (n.kind === "service") shape = "rounded=1;arcSize=50;";
        else if (n.kind === "internet" || n.kind === "user") shape = "ellipse;";
        else if (n.kind === "lb" || n.kind === "external" || n.kind === "aws") shape = "shape=hexagon;perimeter=hexagonPerimeter2;size=0.08;";
        else if (n.kind === "note") shape = "shape=note;size=12;";
        const parent = n.group && gids[n.group] ? gids[n.group] : "1";
        const gb = n.group && groups[n.group];
        const x = Math.round(p.x - (gb ? gb.x : 0)), y = Math.round(p.y - (gb ? gb.y : 0));
        const m = n.meta || {};
        const extra = m.pods != null ? "<br><font style=\"font-size:9px\" color=\"#6b7280\">" + xmlEsc([m.podsReady + "/" + m.pods, m.cpu, m.mem].filter(Boolean).join(" · ")) + "</font>" : "";
        const value = "<b>" + xmlEsc(n.name) + "</b><br><font style=\"font-size:10px\" color=\"#6b7280\">" + xmlEsc(n.sub || KIND_LABEL[n.kind] || "") + "</font>" + extra;
        cells.push('<mxCell id="n' + i + '" value="' + xmlEsc(value) + '" style="' + shape + "whiteSpace=wrap;html=1;fillColor=#ffffff;strokeColor=" + (n.sev === 2 ? "#ef4444" : c) +
          ";strokeWidth=" + (n.sev ? 2 : 1.5) + ';fontColor=#111827;fontSize=12;" vertex="1" parent="' + parent + '"><mxGeometry x="' + x + '" y="' + y + '" width="' + W + '" height="' + H + '" as="geometry"/></mxCell>');
      });
      st.edges.forEach((e, i) => {
        const s = EDGE[e.kind] || EDGE.ref;
        cells.push('<mxCell id="e' + i + '" value="' + xmlEsc(e.label ? trunc(e.label, 40) : "") + '" style="edgeStyle=orthogonalEdgeStyle;rounded=1;html=1;strokeColor=' + s.c +
          ";strokeWidth=" + s.w + ";" + (s.dash ? "dashed=1;" : "") + 'fontSize=10;fontColor=#374151;endArrow=block;endFill=1;" edge="1" parent="1" source="' + ids[e.from] + '" target="' + ids[e.to] + '"><mxGeometry relative="1" as="geometry"/></mxCell>');
      });
      return '<mxfile host="kube-oxente"><diagram name="arquitetura" id="koxente"><mxGraphModel grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="0" pageScale="1" math="0" shadow="0"><root>' +
        cells.join("") + "</root></mxGraphModel></diagram></mxfile>";
    }

    applyView();
    return {
      st: st,
      impact: root => { st.impact = root ? impactOf(root) : null; applyHighlight(); return st.impact; },
      exportMermaid: exportMermaid, exportDrawio: exportDrawio, setData: setData, fit: fit, center: center, select: select, renderAll: renderAll,
      zoom: f => { const r = svg.getBoundingClientRect(); zoomAt(f, r.width / 2, r.height / 2); },
      setConnect: on => { st.connect = on; st.connectFrom = null; svg.classList.toggle("connecting", on); applyHighlight(); },
      setLabels: on => { st.showLabels = on; applyHighlight(); },
      setMatch: set => { st.match = set; applyHighlight(); },
      setView: v => { if (v) { st.view = { x: v.x, y: v.y, k: v.k }; applyView(); } },
      exportSVG: exportSVG, exportPNG: exportPNG, toWorld: toWorld
    };
  }

  const S = {
    mode: "diagram", graph: null, ctx: "", ns: null, lastFetch: 0, loading: false,
    layers: lsGet("koxente.arch.layers", { config: false, storage: true, hpa: true, external: true, netpol: true, labels: false, system: false }),
    dg: null, tg: null, sel: null, connect: false, tf: null, tfSaveT: null, tfParseT: null, searchIdx: 0
  };

  function key() { return "koxente.arch." + (S.ctx || "default"); }
  function tfKey() { return "koxente.tf." + (S.ctx || "default"); }
  function saved() {
    const st = lsGet(key(), { pos: {}, view: null, custom: { nodes: [], edges: [] }, hidden: [], notes: {} });
    st.custom = st.custom || { nodes: [], edges: [] };
    return st;
  }
  function save(st) { lsSet(key(), st); }

  function root() { return document.getElementById("ar-root"); }

  window.rArch = function (m) {
    if (!root()) { mount(m); return; }
    const cs = document.getElementById("ctxsel");
    if (cs && cs.value && S.ctx && cs.value !== S.ctx) {
      S.ctx = cs.value; S.sig = null; S.sel = null; S.graph = null;
      setMode(S.mode);
      return;
    }
    const ns = typeof globalNS !== "undefined" ? globalNS : "";
    if (S.mode === "diagram" && (ns !== S.ns || Date.now() - S.lastFetch > 45000)) fetchGraph(false);
  };

  function mount(m) {
    m.innerHTML =
      '<div class="ar-root" id="ar-root">' +
      '<div class="ar-bar">' +
      "<h2>Arquitetura</h2>" +
      '<div class="ar-seg"><button data-mode="diagram">Diagrama</button><button data-mode="tf">Terraform</button></div>' +
      '<input type="text" class="ar-search" id="ar-q" placeholder="Buscar recurso…">' +
      '<div class="ar-chips" id="ar-chips"></div>' +
      '<span class="ar-sp"></span><span id="ar-tools" style="display:flex;gap:6px;flex-wrap:wrap"></span>' +
      "</div>" +
      '<div class="ar-body" id="ar-body"></div>' +
      "</div>";
    m.querySelectorAll(".ar-seg button").forEach(b => b.onclick = () => setMode(b.getAttribute("data-mode")));
    const q = document.getElementById("ar-q");
    q.oninput = () => { S.searchIdx = 0; runSearch(false); };
    q.onkeydown = ev => { if (ev.key === "Enter") { S.searchIdx++; runSearch(true); } if (ev.key === "Escape") { q.value = ""; runSearch(false); } };
    if (/^#arch:terraform/.test(location.hash)) S.mode = "tf";
    setMode(S.mode);
  }

  function setMode(mode) {
    S.mode = mode;
    S.connect = false;
    try { history.replaceState(null, "", mode === "tf" ? "#arch:terraform" : "#arch"); } catch (e) { }
    const r = root();
    if (!r) return;
    r.querySelectorAll(".ar-seg button").forEach(b => b.classList.toggle("on", b.getAttribute("data-mode") === mode));
    if (mode === "diagram") mountDiagram(); else mountTF();
  }

  function zoomCtl(id) {
    return '<div class="ar-zoom">' +
      '<button data-z="in" title="Aproximar">+</button><button data-z="out" title="Afastar">−</button>' +
      '<button data-z="fit" title="Enquadrar">⊡</button><hr>' +
      '<button data-z="exp" class="sm" title="Exportar (PNG, SVG, Mermaid, draw.io)">⤓</button></div>' +
      '<div class="ar-zl" id="' + id + '-zl"></div>';
  }

  function bindZoom(stage, dg, name) {
    stage.querySelectorAll("[data-z]").forEach(b => b.onclick = () => {
      const z = b.getAttribute("data-z");
      if (z === "in") dg.zoom(1.2);
      if (z === "out") dg.zoom(1 / 1.2);
      if (z === "fit") dg.fit();
      if (z === "exp") openExportMenu(stage, b, dg, name);
    });
  }

  function openExportMenu(stage, btn, dg, name) {
    closePop();
    const m = document.createElement("div");
    m.className = "tf-menu";
    m.id = "tf-menu";
    const br = btn.getBoundingClientRect(), sr = stage.getBoundingClientRect();
    m.style.right = (sr.right - br.left + 8) + "px";
    m.style.top = (br.top - sr.top) + "px";
    const items = [
      ["h", "Imagem"], ["png", "PNG"], ["svg", "SVG"],
      ["h", "Diagrama editável"], ["mmd", "Mermaid (.mmd)"], ["mmdcopy", "Copiar Mermaid"], ["drawio", "draw.io (.drawio)"]
    ];
    m.innerHTML = items.map(i => i[0] === "h" ? '<div class="h">' + i[1] + "</div>" : '<div data-x="' + i[0] + '">' + i[1] + "</div>").join("");
    stage.appendChild(m);
    m.querySelectorAll("[data-x]").forEach(x => x.onclick = () => {
      const k = x.getAttribute("data-x");
      if (k === "png") dg.exportPNG(name());
      if (k === "svg") dg.exportSVG(name());
      if (k === "mmd") download(name() + ".mmd", new Blob([dg.exportMermaid()], { type: "text/plain" }));
      if (k === "mmdcopy" && navigator.clipboard) navigator.clipboard.writeText(dg.exportMermaid()).then(() => toast(stage, "Mermaid copiado — cole num README ou no GitHub"));
      if (k === "drawio") download(name() + ".drawio", new Blob([dg.exportDrawio()], { type: "application/xml" }));
      closePop();
    });
    setTimeout(() => document.addEventListener("pointerdown", function off(ev) {
      if (!m.contains(ev.target)) { closePop(); document.removeEventListener("pointerdown", off); }
    }), 0);
  }

  function toast(stage, msg) {
    const t = document.createElement("div");
    t.className = "ar-hint";
    t.textContent = msg;
    stage.appendChild(t);
    setTimeout(() => t.remove(), 2200);
  }

  function fileBase(suffix) {
    const c = (S.ctx || "cluster").replace(/^arn:aws:eks:[^:]+:\d+:cluster\//, "").replace(/[^a-zA-Z0-9_.-]+/g, "_");
    return "koxente-" + suffix + "-" + c + (S.ns ? "-" + S.ns : "");
  }

  function mountDiagram() {
    const body = document.getElementById("ar-body");
    body.innerHTML = '<div class="ar-stage" id="ar-stage">' + zoomCtl("ar") +
      '<div class="ar-legend">' + LEGEND.map(l => '<span><b class="' + (EDGE[l[0]].dash ? (l[0] === "config" ? "o" : "d") : "") + '" style="border-color:' + EDGE[l[0]].c + '"></b>' + l[1] + "</span>").join("") + "</div>" +
      '<div class="ar-empty" id="ar-empty">lendo o cluster…</div></div>' +
      '<div class="ar-insp" id="ar-insp" style="display:none"></div>';
    const stage = document.getElementById("ar-stage");
    S.dg = createDiagram(stage, {
      key: "dg",
      onZoom: k => { const z = document.getElementById("ar-zl"); if (z) z.textContent = Math.round(k * 100) + "%"; },
      onMove: persistPositions,
      onView: () => { const st = saved(); st.view = S.dg.st.view; save(st); },
      onSelect: s => { S.sel = s; renderInspector(); },
      onConnect: (a, b) => addCustomEdge(a, b),
      onConnectStart: () => hint("agora clique no destino"),
      onDblEmpty: (w, sx, sy) => openAddNode(w, sx, sy),
      onGroupCollapse: gid => setCollapsed(gid, true),
      onDblNode: id => { const n = S.dg.st.byId[id]; if (n && n.collapsedNS) { setCollapsed(n.collapsedNS, false); return; } if (n && n.custom) { S.dg.select({ type: "node", id: id }); setTimeout(() => { const i = document.getElementById("ar-cn-name"); if (i) i.focus(); }, 30); } }
    });
    bindZoom(stage, S.dg, () => fileBase("arquitetura"));
    renderChips();
    renderTools();
    fetchGraph(true);
  }

  function renderChips() {
    const c = document.getElementById("ar-chips");
    if (!c) return;
    if (S.mode !== "diagram") { c.innerHTML = ""; return; }
    c.innerHTML = LAYERS.map(l => '<span class="ar-chip' + (S.layers[l.id] ? " on" : "") + '" data-l="' + l.id + '"><i style="background:' + l.c + '"></i>' + l.label + "</span>").join("");
    c.querySelectorAll("[data-l]").forEach(x => x.onclick = () => {
      const id = x.getAttribute("data-l");
      S.layers[id] = !S.layers[id];
      lsSet("koxente.arch.layers", S.layers);
      renderChips();
      if (id === "system") fetchGraph(false);
      else if (id === "labels") S.dg.setLabels(S.layers.labels);
      else applyGraph(false);
    });
  }

  function renderTools() {
    const t = document.getElementById("ar-tools");
    if (!t) return;
    if (S.mode === "diagram") {
      const hidden = (saved().hidden || []).length;
      t.innerHTML =
        '<button class="ar-btn" id="ar-add" title="Adicionar um nó manual (ou dê duplo clique no canvas)">+ nó</button>' +
        '<button class="ar-btn' + (S.connect ? " on" : "") + '" id="ar-con" title="Ligar dois nós (ou arraste a bolinha roxa de um nó)">⤳ conectar</button>' +
        '<button class="ar-btn" id="ar-lay" title="Reorganizar automaticamente">organizar</button>' +
        '<button class="ar-btn" id="ar-col" title="Recolher ou expandir todos os namespaces">' + ((saved().collapsed || []).length ? "expandir tudo" : "recolher tudo") + "</button>" +
        (hidden ? '<button class="ar-btn" id="ar-unh" title="Mostrar nós ocultados">ocultos (' + hidden + ")</button>" : "") +
        '<button class="ar-btn" id="ar-ref" title="Ler o cluster de novo">↻ cluster</button>';
      document.getElementById("ar-add").onclick = () => {
        const st = document.getElementById("ar-stage").getBoundingClientRect();
        const w = S.dg.toWorld(st.left + st.width / 2, st.top + st.height / 2);
        openAddNode(w, st.width / 2 - 130, 60);
      };
      document.getElementById("ar-con").onclick = () => {
        S.connect = !S.connect;
        S.dg.setConnect(S.connect);
        renderTools();
        if (S.connect) hint("clique na origem e depois no destino"); else hint("");
      };
      document.getElementById("ar-lay").onclick = () => { const st = saved(); st.pos = {}; save(st); applyGraph(true); setTimeout(() => S.dg.fit(), 30); };
      document.getElementById("ar-col").onclick = () => {
        const st = saved();
        st.collapsed = (st.collapsed || []).length ? [] : allNamespaces();
        save(st);
        S.sel = null;
        S.dg.impact(null);
        applyGraph(false);
        setTimeout(() => S.dg.fit(), 30);
      };
      const u = document.getElementById("ar-unh");
      if (u) u.onclick = () => { const st = saved(); st.hidden = []; save(st); renderTools(); applyGraph(false); };
      document.getElementById("ar-ref").onclick = () => fetchGraph(false, true);
    } else {
      t.innerHTML =
        '<button class="ar-btn" id="tf-add">+ recurso</button>' +
        '<button class="ar-btn" id="tf-lay">organizar</button>' +
        '<button class="ar-btn" id="tf-copy">copiar</button>' +
        '<button class="ar-btn" id="tf-dl">baixar .tf</button>' +
        '<button class="ar-btn" id="tf-regen" title="Gerar o HCL de novo a partir do cluster">↻ gerar do cluster</button>';
      document.getElementById("tf-add").onclick = ev => {
        const r = document.getElementById("tf-stage").getBoundingClientRect();
        const w = S.tg.toWorld(r.left + r.width / 2, r.top + r.height / 2);
        const br = ev.target.getBoundingClientRect(), sr = document.getElementById("ar-body").getBoundingClientRect();
        openTFMenu(w, br.left - sr.left - 120, 4, true);
      };
      document.getElementById("tf-lay").onclick = () => { const s = tfState(); s.pos = {}; lsSet(tfKey(), s); parseAndDraw(true); setTimeout(() => S.tg.fit(), 30); };
      document.getElementById("tf-copy").onclick = () => { navigator.clipboard && navigator.clipboard.writeText(tfText()); flashStatus("copiado"); };
      document.getElementById("tf-dl").onclick = () => download(fileBase("terraform") + ".tf", new Blob([tfText()], { type: "text/plain" }));
      document.getElementById("tf-regen").onclick = () => {
        if (S.tfDirty && !confirm("Substituir o HCL editado pelo gerado a partir do cluster?")) return;
        loadTFFromCluster();
      };
    }
  }

  function hint(msg) {
    const st = document.getElementById("ar-stage") || document.getElementById("tf-stage");
    if (!st) return;
    let e = st.querySelector(".ar-hint");
    if (!msg) { if (e) e.remove(); return; }
    if (!e) { e = document.createElement("div"); e.className = "ar-hint"; st.appendChild(e); }
    e.textContent = msg;
  }

  function fetchGraph(first, force) {
    const ns = typeof globalNS !== "undefined" ? globalNS : "";
    if (S.loading) return;
    S.loading = true;
    S.ns = ns;
    const empty = document.getElementById("ar-empty");
    if (force && empty) { empty.style.display = ""; empty.textContent = "lendo o cluster…"; }
    fetch("/api/arch?ns=" + encodeURIComponent(ns) + (S.layers.system ? "&system=1" : ""))
      .then(r => r.json())
      .then(g => {
        S.loading = false;
        S.lastFetch = Date.now();
        const sig = JSON.stringify([g.nodes.map(n => n.id + n.sev + n.sub), g.edges.map(e => e.id)]);
        const changed = sig !== S.sig || g.context !== S.ctx;
        S.sig = sig;
        const ctxChanged = g.context !== S.ctx;
        S.ctx = g.context;
        S.graph = g;
        if (typeof setGen === "function") setGen();
        if (!S.dg || S.mode !== "diagram") return;
        if (changed || first || force) {
          const st = saved();
          if (st.collapsed === undefined) {
            const nsl = allNamespaces();
            st.collapsed = nsl.length > 6 && g.nodes.length > 150 ? nsl : [];
            save(st);
          }
          applyGraph(false);
          if (first || ctxChanged) { if (st.view && !ctxChanged) S.dg.setView(st.view); else setTimeout(() => S.dg.fit(), 20); }
        }
      })
      .catch(err => {
        S.loading = false;
        const e = document.getElementById("ar-empty");
        if (e) { e.style.display = ""; e.textContent = "erro ao ler o cluster: " + err; }
      });
  }

  function visibleGraph() {
    const g = S.graph || { nodes: [], edges: [] };
    const st = saved();
    const hidden = new Set(st.hidden || []);
    const off = new Set();
    LAYERS.forEach(l => { if (!S.layers[l.id]) l.kinds.forEach(k => off.add(k)); });
    const nodes = g.nodes.filter(n => !off.has(n.kind) && !hidden.has(n.id)).map(n => Object.assign({}, n));
    const ids = new Set(nodes.map(n => n.id));
    let edges = g.edges.filter(e => ids.has(e.from) && ids.has(e.to) && (S.layers.netpol || e.kind !== "netpol"));
    const deg = {};
    edges.forEach(e => { deg[e.from] = (deg[e.from] || 0) + 1; deg[e.to] = (deg[e.to] || 0) + 1; });
    const out = nodes.filter(n => n.kind !== "internet" || deg[n.id]);
    (st.custom.nodes || []).forEach(n => { if (!hidden.has(n.id)) out.push(Object.assign({ custom: true }, n)); });
    (st.custom.edges || []).forEach(e => edges.push(Object.assign({ custom: true, kind: "custom" }, e)));
    return collapseGroups(out, edges, new Set(st.collapsed || []));
  }

  function collapseGroups(nodes, edges, collapsed) {
    if (!collapsed.size) return { nodes: nodes, edges: edges };
    const map = {}, agg = {};
    const out = [];
    nodes.forEach(n => {
      if (!n.group || !collapsed.has(n.group)) { out.push(n); return; }
      const gid = "nsgrp:" + n.group;
      map[n.id] = gid;
      const a = agg[gid] = agg[gid] || { id: gid, kind: "namespace", name: n.group, ns: n.group, collapsedNS: n.group, sev: 0, count: 0, bad: 0, wl: 0 };
      a.count++;
      if (n.kind === "workload") a.wl++;
      if (n.sev) a.bad++;
      a.sev = Math.max(a.sev, n.sev || 0);
    });
    Object.keys(agg).forEach(gid => {
      const a = agg[gid];
      a.sub = a.wl + " workloads · " + a.count + " recursos" + (a.bad ? " · " + a.bad + " com alerta" : "");
      out.push(a);
    });
    const seen = new Set();
    const outE = [];
    edges.forEach(e => {
      const f = map[e.from] || e.from, t = map[e.to] || e.to;
      if (f === t) return;
      const id = f + ">" + t + ">" + e.kind;
      if (seen.has(id)) return;
      seen.add(id);
      outE.push(Object.assign({}, e, { id: e.custom ? e.id : id, from: f, to: t, label: map[e.from] || map[e.to] ? "" : e.label }));
    });
    return { nodes: out, edges: outE };
  }

  function setCollapsed(ns, on) {
    const st = saved();
    const set = new Set(st.collapsed || []);
    if (on) set.add(ns); else set.delete(ns);
    st.collapsed = Array.from(set);
    save(st);
    S.sel = null;
    if (S.dg) { S.dg.impact(null); S.dg.select(null); }
    applyGraph(false);
  }

  function allNamespaces() {
    const set = new Set();
    ((S.graph && S.graph.nodes) || []).forEach(n => { if (n.group) set.add(n.group); });
    return Array.from(set);
  }

  function applyGraph(relayout) {
    if (!S.dg) return;
    const vg = visibleGraph();
    const st = saved();
    S.dg.setData(vg.nodes, vg.edges, st.pos, relayout);
    S.dg.setLabels(S.layers.labels);
    if (relayout) persistPositions();
    const empty = document.getElementById("ar-empty");
    if (empty && S.graph && S.graph.error) {
      empty.style.display = "";
      empty.textContent = "não consegui ler o cluster (contexto " + S.graph.context + "): " + S.graph.error;
    } else if (empty) {
      if (!vg.nodes.length) { empty.style.display = ""; empty.textContent = "nada para desenhar aqui — troque o namespace ou ative \"Sistema\"."; }
      else empty.style.display = "none";
    }
    renderTools();
    if (S.sel) renderInspector();
  }

  function persistPositions() {
    const st = saved();
    st.pos = Object.assign(st.pos || {}, S.dg.st.pos);
    save(st);
  }

  function addCustomEdge(a, b) {
    const st = saved();
    const id = "custom:e:" + Date.now().toString(36);
    st.custom.edges.push({ id: id, from: a, to: b, label: "" });
    save(st);
    if (S.connect) { S.connect = false; S.dg.setConnect(false); renderTools(); }
    hint("");
    applyGraph(false);
    S.dg.select({ type: "edge", id: id });
  }

  function openAddNode(w, sx, sy) {
    closePop();
    const kinds = ["external", "db", "cache", "queue", "storage", "aws", "service", "workload", "lb", "user", "internet", "security", "note"];
    const p = document.createElement("div");
    p.className = "ar-pop";
    p.id = "ar-pop";
    p.style.left = Math.max(8, sx) + "px";
    p.style.top = Math.max(8, sy) + "px";
    p.innerHTML = "<b>Novo nó</b>" +
      "<label>Tipo</label><select id='ar-nk'>" + kinds.map(k => "<option value='" + k + "'>" + KIND_LABEL[k] + "</option>").join("") + "</select>" +
      "<label>Nome</label><input id='ar-nn' type='text' placeholder='ex.: RDS pagamentos'>" +
      "<label>Descrição</label><input id='ar-ns' type='text' placeholder='ex.: postgres 16 · sa-east-1'>" +
      "<div class='row'><button id='ar-nc'>cancelar</button><button id='ar-no' class='btn-live'>adicionar</button></div>";
    document.getElementById("ar-stage").appendChild(p);
    const nn = document.getElementById("ar-nn");
    nn.focus();
    const ok = () => {
      const name = nn.value.trim();
      if (!name) { nn.focus(); return; }
      const st = saved();
      const id = "custom:n:" + Date.now().toString(36);
      const kind = document.getElementById("ar-nk").value;
      st.custom.nodes.push({ id: id, kind: kind, name: name, sub: document.getElementById("ar-ns").value.trim() || KIND_LABEL[kind] });
      st.pos[id] = { x: Math.round(w.x - W / 2), y: Math.round(w.y - H / 2) };
      save(st);
      closePop();
      applyGraph(false);
      S.dg.select({ type: "node", id: id });
    };
    document.getElementById("ar-no").onclick = ok;
    document.getElementById("ar-nc").onclick = closePop;
    p.addEventListener("keydown", ev => { if (ev.key === "Enter") ok(); if (ev.key === "Escape") closePop(); });
  }

  function closePop() { const p = document.getElementById("ar-pop"); if (p) p.remove(); const m = document.getElementById("tf-menu"); if (m) m.remove(); }

  function renderInspector() {
    const box = document.getElementById("ar-insp");
    if (!box || !S.dg) return;
    const s = S.sel;
    if (!s) { box.style.display = "none"; return; }
    const st = saved();
    const dst = S.dg.st;
    if (s.type === "edge") {
      const e = dst.edges.find(x => x.id === s.id);
      if (!e) { box.style.display = "none"; return; }
      const a = dst.byId[e.from], b = dst.byId[e.to];
      box.style.display = "";
      box.innerHTML = '<span class="x" id="ar-ix">×</span><h3>Conexão</h3>' +
        '<div class="muted">' + h(a ? a.name : e.from) + " → " + h(b ? b.name : e.to) + "</div>" +
        '<div class="k">Tipo</div><div>' + h(e.custom ? "manual" : e.kind) + "</div>" +
        (e.custom ? '<div class="k">Rótulo</div><input id="ar-el" type="text" value="' + h(e.label || "") + '" placeholder="ex.: HTTPS 443, gRPC, SQL">' +
          '<div class="row"><button id="ar-ed" class="btn-stop">remover conexão</button></div>'
          : (e.label ? '<div class="k">Detalhe</div><pre>' + h(e.label) + "</pre>" : "") +
          ((e.evidence || []).length ? '<div class="k">Por que existe</div><pre>' + h(e.evidence.join("\n")) + "</pre>" : ""));
      document.getElementById("ar-ix").onclick = () => S.dg.select(null);
      const el1 = document.getElementById("ar-el");
      if (el1) el1.onchange = () => { const st2 = saved(); const ce = st2.custom.edges.find(x => x.id === e.id); if (ce) { ce.label = el1.value.trim(); save(st2); applyGraph(false); } };
      const ed = document.getElementById("ar-ed");
      if (ed) ed.onclick = () => deleteSelected();
      return;
    }
    const n = dst.byId[s.id];
    if (!n) { box.style.display = "none"; return; }
    box.style.display = "";
    const conns = dst.edges.filter(e => e.from === n.id || e.to === n.id);
    const meta = n.meta || {};
    let html = '<span class="x" id="ar-ix">×</span>';
    if (n.collapsedNS) {
      box.innerHTML = html + "<h3>" + h(n.name) + '</h3><div class="muted">namespace recolhido · ' + h(n.sub) + "</div>" +
        '<div class="row"><button id="ar-iexp" class="btn-live">expandir</button><button id="ar-iimp">impacto</button></div>' + impactHTML(n.id);
      document.getElementById("ar-ix").onclick = () => S.dg.select(null);
      document.getElementById("ar-iexp").onclick = () => setCollapsed(n.collapsedNS, false);
      document.getElementById("ar-iimp").onclick = () => toggleImpact(n.id);
      bindImpactList(box);
      return;
    }
    if (n.custom) {
      html += "<h3>" + h(n.name) + '</h3><div class="muted">nó manual</div>' +
        '<div class="k">Nome</div><input id="ar-cn-name" type="text" value="' + h(n.name) + '">' +
        '<div class="k">Descrição</div><input id="ar-cn-sub" type="text" value="' + h(n.sub || "") + '">' +
        '<div class="k">Tipo</div><select id="ar-cn-kind">' + Object.keys(KIND_LABEL).filter(k => k !== "tf" && k !== "group").map(k => "<option value='" + k + "'" + (k === n.kind ? " selected" : "") + ">" + KIND_LABEL[k] + "</option>").join("") + "</select>";
    } else {
      html += "<h3>" + h(n.name) + "</h3>" +
        '<div class="muted">' + h(n.sub || KIND_LABEL[n.kind]) + "</div>" +
        (n.ns ? '<div class="k">Namespace</div><div>' + h(n.ns) + ' <span class="badge ' + envOf(n.ns) + '">' + envOf(n.ns) + "</span></div>" : "") +
        healthHTML(n) +
        (meta.images ? '<div class="k">Imagens</div><pre>' + h(meta.images) + "</pre>" : "") +
        (meta.serviceAccount ? '<div class="k">ServiceAccount</div><div class="mono">' + h(meta.serviceAccount) + "</div>" : "") +
        (meta.hosts ? '<div class="k">Hosts</div><pre>' + h(meta.hosts) + "</pre>" : "") +
        (meta.host ? '<div class="k">Endereço</div><pre>' + h(meta.host) + "</pre>" : "") +
        (meta.albGroup ? '<div class="k">ALB group</div><div class="mono">' + h(meta.albGroup) + "</div>" : "");
    }
    html += '<div class="k">Conexões (' + conns.length + ")</div>" + (conns.length ? conns.map(e => {
      const out = e.from === n.id;
      const o = dst.byId[out ? e.to : e.from];
      return '<div class="cx" data-go="' + h(o ? o.id : "") + '"><span>' + (out ? "→" : "←") + "</span><div>" + h(o ? o.name : "?") +
        "<small>" + h((e.custom ? "manual" : e.kind) + (e.label ? " · " + e.label : "")) + "</small>" +
        ((e.evidence || []).length ? "<small>" + h(e.evidence.join(" · ")) + "</small>" : "") + "</div></div>";
    }).join("") : '<div class="muted">nenhuma</div>');
    html += impactHTML(n.id);
    html += '<div class="k">Anotações</div><textarea id="ar-note" placeholder="observações suas sobre este recurso (fica salvo neste navegador)">' + h((st.notes || {})[n.id] || "") + "</textarea>";
    html += '<div class="row">' +
      '<button id="ar-ic">centralizar</button>' +
      '<button id="ar-iimp" title="Quem é afetado se este recurso cair (atalho: i)">' + (S.dg.st.impact && S.dg.st.impact.root === n.id ? "limpar impacto" : "impacto") + "</button>" +
      (n.kind === "workload" ? '<button id="ar-ip">ver pods</button>' : "") +
      (n.custom ? '<button id="ar-idel" class="btn-stop">remover</button>' : '<button id="ar-ih">ocultar</button>') + "</div>";
    box.innerHTML = html;
    document.getElementById("ar-ix").onclick = () => S.dg.select(null);
    document.getElementById("ar-ic").onclick = () => S.dg.center(n.id);
    document.getElementById("ar-iimp").onclick = () => toggleImpact(n.id);
    bindImpactList(box);
    const idg = document.getElementById("ar-idiag");
    if (idg) idg.onclick = () => { if (typeof openDetail === "function") openDetail(n.ns, meta.samplePod, meta.reason || ""); };
    if (meta.samplePod && n.sev) loadProblemHint(n);
    box.querySelectorAll("[data-go]").forEach(x => x.onclick = () => { const id = x.getAttribute("data-go"); if (id) { S.dg.center(id); S.dg.select({ type: "node", id: id }); } });
    document.getElementById("ar-note").onchange = ev => { const st2 = saved(); st2.notes = st2.notes || {}; st2.notes[n.id] = ev.target.value; save(st2); };
    const ip = document.getElementById("ar-ip");
    if (ip) ip.onclick = () => {
      if (typeof filt !== "undefined") { filt.q = n.name; filt.app = ""; }
      if (typeof nav === "function") nav("pods");
    };
    const ih = document.getElementById("ar-ih");
    if (ih) ih.onclick = () => deleteSelected();
    const idel = document.getElementById("ar-idel");
    if (idel) idel.onclick = () => deleteSelected();
    if (n.custom) {
      const upd = () => {
        const st2 = saved();
        const cn = st2.custom.nodes.find(x => x.id === n.id);
        if (!cn) return;
        cn.name = document.getElementById("ar-cn-name").value.trim() || cn.name;
        cn.sub = document.getElementById("ar-cn-sub").value.trim();
        cn.kind = document.getElementById("ar-cn-kind").value;
        save(st2);
        applyGraph(false);
      };
      ["ar-cn-name", "ar-cn-sub", "ar-cn-kind"].forEach(i => document.getElementById(i).onchange = upd);
    }
  }

  function healthHTML(n) {
    const m = n.meta || {};
    if (m.pods == null) return "";
    const ready = +m.podsReady, total = +m.pods;
    const cls = total && ready < total ? (ready ? "color:var(--yel)" : "color:var(--red)") : "color:var(--grn)";
    let out = '<div class="k">Saúde agora</div><div class="ar-hgrid">' +
      '<div><b style="' + cls + '">' + ready + "/" + total + "</b><small>pods prontos</small></div>" +
      "<div><b>" + h(m.cpu || "–") + "</b><small>CPU</small></div>" +
      "<div><b>" + h(m.mem || "–") + "</b><small>memória</small></div>" +
      '<div><b style="' + (+m.restarts ? "color:var(--yel)" : "") + '">' + h(m.restarts || "0") + "</b><small>restarts</small></div></div>";
    const anyMetrics = ((S.graph && S.graph.nodes) || []).some(x => x.meta && x.meta.cpu);
    if (!anyMetrics) out += '<div class="muted" style="font-size:11px;margin-top:4px">sem métricas de CPU/memória — o cluster tem metrics-server?</div>';
    if (m.samplePod && n.sev) {
      out += '<div class="ar-prob"><b>' + h(m.reason || "problema") + "</b> em <span class=\"mono\">" + h(m.samplePod) + "</span>" +
        '<div id="ar-phint" class="muted" style="margin-top:4px"></div>' +
        '<div class="row"><button id="ar-idiag" class="btn-stop">ver diagnóstico, eventos e logs</button></div></div>';
    }
    return out;
  }

  function loadProblemHint(n) {
    const pods = new Set(((n.meta || {}).podNames || "").split(",").filter(Boolean));
    fetch("/api/problems").then(r => r.json()).then(list => {
      const el2 = document.getElementById("ar-phint");
      if (!el2 || !S.sel || S.sel.id !== n.id) return;
      const hit = (list || []).find(p => p.ns === n.ns && (p.pods || []).some(x => pods.has(x)));
      const diag = hit && hit.primaryDiag;
      if (diag && diag.toLowerCase() !== String((n.meta || {}).reason || "").toLowerCase()) el2.textContent = "provável causa: " + diag;
    }).catch(() => { });
  }

  function toggleImpact(id) {
    if (!S.dg) return;
    const cur = S.dg.st.impact;
    S.dg.impact(cur && cur.root === id ? null : id);
    renderInspector();
  }

  function impactHTML(id) {
    const imp = S.dg && S.dg.st.impact;
    if (!imp || imp.root !== id) return "";
    const dst = S.dg.st;
    const hit = Array.from(imp.nodes).filter(x => x !== id).map(x => dst.byId[x]).filter(Boolean);
    if (!hit.length) return '<div class="ar-imp-box">Nada no desenho depende deste recurso.</div>';
    const order = ["internet", "lb", "ingress", "service", "workload", "namespace"];
    hit.sort((a, b) => (order.indexOf(a.kind) + 10) % 20 - (order.indexOf(b.kind) + 10) % 20 || a.name.localeCompare(b.name));
    return '<div class="ar-imp-box"><b>Se cair, afeta ' + hit.length + " recurso" + (hit.length > 1 ? "s" : "") + "</b>" +
      hit.map(x => '<div class="cx" data-go="' + h(x.id) + '"><span>↳</span><div>' + h(x.name) + "<small>" + h((x.ns ? x.ns + " · " : "") + (KIND_LABEL[x.kind] || x.kind)) + "</small></div></div>").join("") + "</div>";
  }

  function bindImpactList(box) {
    box.querySelectorAll(".ar-imp-box [data-go]").forEach(x => x.onclick = () => S.dg.center(x.getAttribute("data-go")));
  }

  function deleteSelected() {
    const s = S.sel;
    if (!s || !S.dg) return;
    const st = saved();
    if (s.type === "edge") {
      const before = st.custom.edges.length;
      st.custom.edges = st.custom.edges.filter(e => e.id !== s.id);
      if (st.custom.edges.length === before) return;
    } else {
      const n = S.dg.st.byId[s.id];
      if (!n) return;
      if (n.custom) {
        st.custom.nodes = st.custom.nodes.filter(x => x.id !== n.id);
        st.custom.edges = st.custom.edges.filter(e => e.from !== n.id && e.to !== n.id);
      } else {
        st.hidden = (st.hidden || []).concat([n.id]);
      }
    }
    save(st);
    S.sel = null;
    S.dg.select(null);
    applyGraph(false);
  }

  function runSearch(center) {
    const q = (document.getElementById("ar-q") || {}).value || "";
    const dg = S.mode === "diagram" ? S.dg : S.tg;
    if (!dg) return;
    if (!q.trim()) { dg.setMatch(null); return; }
    const ql = q.toLowerCase();
    const hits = dg.st.nodes.filter(n => (n.name + " " + (n.ns || "") + " " + (n.sub || "") + " " + (n.addr || "")).toLowerCase().indexOf(ql) >= 0);
    dg.setMatch(new Set(hits.map(n => n.id)));
    if (hits.length && (center || hits.length === 1)) dg.center(hits[S.searchIdx % hits.length].id);
  }

  window.addEventListener("hashchange", () => {
    if (!root() || !/^#arch/.test(location.hash)) return;
    const want = /^#arch:terraform/.test(location.hash) ? "tf" : "diagram";
    if (want !== S.mode) setMode(want);
  });

  document.addEventListener("keydown", ev => {
    if (!root()) return;
    const tag = (ev.target.tagName || "").toLowerCase();
    if (tag === "input" || tag === "textarea" || tag === "select") return;
    if (ev.key === "Escape") {
      closePop();
      if (S.connect) { S.connect = false; if (S.dg) S.dg.setConnect(false); renderTools(); hint(""); }
      else if (S.dg && S.mode === "diagram" && S.dg.st.impact) { S.dg.impact(null); renderInspector(); }
      else if (S.dg && S.mode === "diagram") S.dg.select(null);
    }
    if (ev.key === "i" && S.mode === "diagram" && S.sel && S.sel.type === "node") toggleImpact(S.sel.id);
    if ((ev.key === "Delete" || ev.key === "Backspace") && S.mode === "diagram" && S.sel) { ev.preventDefault(); deleteSelected(); }
  });

  function tfState() { return lsGet(tfKey(), { text: null, pos: {}, view: null, dirty: false }); }
  function tfText() { const ta = document.getElementById("tf-ta"); return ta ? ta.value : (tfState().text || ""); }

  function mountTF() {
    const body = document.getElementById("ar-body");
    body.innerHTML =
      '<div class="tf-wrap">' +
      '<div class="tf-left">' +
      '<div class="tf-tabs"><span class="tf-tab on"><span class="dot">●</span>main.tf<span class="err" id="tf-errc"></span></span>' +
      '<span class="tf-st"><span id="tf-cnt"></span><span id="tf-saved" class="g"></span></span></div>' +
      '<div class="tf-ed" id="tf-ed"><div class="tf-gut" id="tf-gut"></div><div class="tf-code"><pre id="tf-pre"></pre><textarea id="tf-ta" spellcheck="false" wrap="off" autocapitalize="off" autocomplete="off"></textarea></div></div>' +
      '<div class="tf-bottom"><div class="tf-btabs" id="tf-btabs"></div><div class="tf-bpanel" id="tf-bpanel"></div></div>' +
      "</div>" +
      '<div class="ar-stage" id="tf-stage">' + zoomCtl("tf") + '<div class="ar-empty" id="tf-empty">gerando a partir do cluster…</div></div>' +
      "</div>";
    if (!S.ctx) {
      fetch("/api/contexts").then(r => r.json()).then(d => { S.ctx = d.active || ""; initTF(); });
    } else initTF();
  }

  function initTF() {
    const stage = document.getElementById("tf-stage");
    if (!stage) return;
    S.tg = createDiagram(stage, {
      key: "tg",
      onZoom: k => { const z = document.getElementById("tf-zl"); if (z) z.textContent = Math.round(k * 100) + "%"; },
      onMove: () => { const s = tfState(); s.pos = Object.assign(s.pos || {}, S.tg.st.pos); lsSet(tfKey(), s); },
      onView: () => { const s = tfState(); s.view = S.tg.st.view; lsSet(tfKey(), s); },
      onSelect: s => { if (s && s.type === "node") { const n = S.tg.st.byId[s.id]; if (n && n.line) gotoLine(n.line); } },
      onConnect: (a, b) => tfAddDepends(a, b),
      onDblEmpty: (w, sx, sy) => openTFMenu(w, sx, sy, false),
      onDblNode: id => { const n = S.tg.st.byId[id]; if (n && n.line) gotoLine(n.line, true); },
      groupLabel: gid => (S.tf && S.tf.groupLabels[gid]) || gid,
      groupTag: gid => (S.tf && S.tf.groupTags[gid]) || "namespace",
      groupColor: gid => gid.indexOf("vpc:") === 0 ? COLORS.network : null
    });
    bindZoom(stage, S.tg, () => fileBase("terraform"));
    renderChips();
    renderTools();
    const ta = document.getElementById("tf-ta");
    ta.addEventListener("input", () => {
      S.tfDirty = true;
      paintEditor();
      clearTimeout(S.tfParseT);
      S.tfParseT = setTimeout(() => parseAndDraw(false), 250);
      clearTimeout(S.tfSaveT);
      document.getElementById("tf-saved").textContent = "";
      S.tfSaveT = setTimeout(() => { const s = tfState(); s.text = ta.value; s.dirty = true; lsSet(tfKey(), s); flashStatus("salvo"); }, 600);
    });
    ta.addEventListener("keydown", ev => {
      if (ev.key === "Tab") {
        ev.preventDefault();
        const s = ta.selectionStart, e = ta.selectionEnd;
        ta.value = ta.value.slice(0, s) + "  " + ta.value.slice(e);
        ta.selectionStart = ta.selectionEnd = s + 2;
        ta.dispatchEvent(new Event("input"));
      }
    });
    const s = tfState();
    if (s.text) {
      S.tfDirty = !!s.dirty;
      setTFText(s.text);
      if (s.view) S.tg.setView(s.view); else setTimeout(() => S.tg.fit(), 30);
      flashStatus(s.dirty ? "salvo (editado)" : "salvo");
    } else loadTFFromCluster();
  }

  function loadTFFromCluster() {
    const empty = document.getElementById("tf-empty");
    if (empty) { empty.style.display = ""; empty.textContent = "gerando a partir do cluster…"; }
    const ns = typeof globalNS !== "undefined" ? globalNS : "";
    S.ns = ns;
    fetch("/api/arch/terraform?ns=" + encodeURIComponent(ns) + (S.layers.system ? "&system=1" : ""))
      .then(r => r.text())
      .then(txt => {
        const s = tfState();
        s.text = txt; s.dirty = false; s.pos = {};
        lsSet(tfKey(), s);
        S.tfDirty = false;
        setTFText(txt, true);
        setTimeout(() => S.tg.fit(), 30);
        flashStatus("gerado do cluster");
      })
      .catch(err => { if (empty) empty.textContent = "erro: " + err; });
  }

  function setTFText(txt, relayout) {
    const ta = document.getElementById("tf-ta");
    if (!ta) return;
    ta.value = txt;
    paintEditor();
    parseAndDraw(!!relayout);
  }

  function flashStatus(msg) { const e = document.getElementById("tf-saved"); if (e) e.textContent = msg; }

  function paintEditor() {
    const ta = document.getElementById("tf-ta");
    if (!ta) return;
    document.getElementById("tf-pre").innerHTML = highlight(ta.value) + "\n ";
    const lines = ta.value.split("\n").length;
    const errLines = new Set(((S.tf && S.tf.errors) || []).filter(e => e.sev === "e").map(e => e.line));
    let g = "";
    for (let i = 1; i <= lines; i++) g += (errLines.has(i) ? '<span class="e">' + i + "</span>" : i) + "\n";
    document.getElementById("tf-gut").innerHTML = g;
  }

  function gotoLine(line, focus) {
    const ta = document.getElementById("tf-ta"), ed = document.getElementById("tf-ed");
    if (!ta || !ed) return;
    const lines = ta.value.split("\n");
    let off = 0;
    for (let i = 0; i < line - 1 && i < lines.length; i++) off += lines[i].length + 1;
    const lh = parseFloat(getComputedStyle(ta).lineHeight) || 20;
    ed.scrollTop = Math.max(0, (line - 4) * lh);
    if (focus) { ta.focus({ preventScroll: true }); ta.setSelectionRange(off, off + (lines[line - 1] || "").length); }
  }

  const KW = /^(resource|data|variable|output|provider|terraform|module|locals|import|moved|removed|check)$/;

  function highlight(text) {
    const out = [];
    let heredoc = null;
    text.split("\n").forEach(line => {
      if (heredoc) {
        out.push('<span class="hl-s">' + h(line) + "</span>");
        if (line.trim() === heredoc) heredoc = null;
        return;
      }
      const hm = line.match(/<<-?([A-Z_]+)\s*$/);
      let res = "";
      const re = /(#.*$|\/\/.*$)|("(?:[^"\\]|\\.)*"?)|\b((?:data\.)?[a-z][a-z0-9]*_[a-z0-9_]+\.[a-zA-Z_][\w-]*|var\.[\w-]+|local\.[\w-]+|module\.[\w-]+)\b|\b(true|false|null|for_each|count|depends_on|lifecycle|dynamic)\b|\b(\d+(?:\.\d+)?)\b|^(\s*)([a-z_][a-z0-9_]*)(?=(?:\s+"[^"]*")*\s*\{)/g;
      let last = 0, m;
      while ((m = re.exec(line))) {
        res += h(line.slice(last, m.index));
        if (m[1]) res += '<span class="hl-c">' + h(m[1]) + "</span>";
        else if (m[2]) res += '<span class="hl-s">' + h(m[2]) + "</span>";
        else if (m[3]) res += '<span class="hl-r">' + h(m[3]) + "</span>";
        else if (m[4]) res += '<span class="hl-k">' + h(m[4]) + "</span>";
        else if (m[5]) res += '<span class="hl-n">' + h(m[5]) + "</span>";
        else if (m[7]) res += h(m[6]) + '<span class="' + (KW.test(m[7]) ? "hl-k" : "hl-b") + '">' + h(m[7]) + "</span>";
        last = re.lastIndex;
        if (m[0] === "") re.lastIndex++;
      }
      res += h(line.slice(last));
      out.push(res);
      if (hm) heredoc = hm[1];
    });
    return out.join("\n");
  }

  function stripCode(text) {
    let out = "";
    let i = 0;
    const n = text.length;
    while (i < n) {
      const c = text[i];
      if (c === "#" || (c === "/" && text[i + 1] === "/")) {
        while (i < n && text[i] !== "\n") { out += " "; i++; }
        continue;
      }
      if (c === "/" && text[i + 1] === "*") {
        while (i < n && !(text[i] === "*" && text[i + 1] === "/")) { out += text[i] === "\n" ? "\n" : " "; i++; }
        out += "  "; i += 2;
        continue;
      }
      if (c === "<" && text[i + 1] === "<") {
        const m = text.slice(i).match(/^<<-?([A-Za-z_]+)[ \t]*\n/);
        if (m) {
          const tag = m[1];
          let j = i + m[0].length;
          out += " ".repeat(m[0].length - 1) + "\n";
          while (j < n) {
            const e = text.indexOf("\n", j);
            const ln = text.slice(j, e < 0 ? n : e);
            out += " ".repeat(ln.length);
            j = e < 0 ? n : e;
            if (ln.trim() === tag) break;
            if (j < n) { out += "\n"; j++; }
          }
          i = j;
          continue;
        }
      }
      if (c === '"') {
        out += " "; i++;
        while (i < n && text[i] !== '"' && text[i] !== "\n") {
          if (text[i] === "\\") { out += "  "; i += 2; continue; }
          if ((text[i] === "$" || text[i] === "%") && text[i + 1] === "{" && text[i - 1] !== text[i]) {
            out += " {"; i += 2;
            let depth = 1;
            while (i < n && depth > 0 && text[i] !== "\n") {
              if (text[i] === "{") depth++;
              if (text[i] === "}") depth--;
              out += depth === 0 ? "}" : (text[i] === '"' ? " " : text[i]);
              i++;
            }
            continue;
          }
          out += " "; i++;
        }
        if (text[i] === '"') { out += " "; i++; }
        continue;
      }
      out += c; i++;
    }
    return out;
  }

  const BUILTIN = new Set(["var", "local", "module", "data", "each", "count", "path", "terraform", "self"]);

  function tfKind(type) {
    const t = type;
    if (/deployment|stateful_set|daemon_set|cron_job|_job_v|replication_controller/.test(t) && /^kubernetes_/.test(t)) return "workload";
    if (/^kubernetes_service(_v1)?$/.test(t)) return "service";
    if (/^kubernetes_ingress/.test(t)) return "ingress";
    if (/^kubernetes_config_map/.test(t)) return "configmap";
    if (/^kubernetes_secret/.test(t)) return "secret";
    if (/persistent_volume/.test(t)) return "pvc";
    if (/horizontal_pod_autoscaler/.test(t)) return "hpa";
    if (/network_policy|security_group|_iam_|_waf|_kms_/.test(t)) return "security";
    if (/_db_|_rds_|docdb|dynamodb|neptune|redshift|mongodbatlas|_sql_/.test(t)) return "db";
    if (/elasticache|memorydb|redis/.test(t)) return "cache";
    if (/_sqs_|_sns_|_mq_|_msk_|kafka|eventbridge|_kinesis/.test(t)) return "queue";
    if (/_s3_|_efs_|_ebs_|storage/.test(t)) return "storage";
    if (/opensearch|elasticsearch/.test(t)) return "search";
    if (/_lb$|_lb_|_alb|_elb|cloudfront|api_gateway|_apigatewayv2/.test(t)) return "lb";
    if (/_vpc|_subnet|_route|_nat_|internet_gateway|_eip|_network/.test(t)) return "network";
    if (/_instance$|_eks_|_ecs_|_lambda_|autoscaling|launch_template|_node_group/.test(t)) return "compute";
    if (/^kubernetes_namespace/.test(t)) return "group";
    return "tf";
  }

  function tfAbbr(t) {
    if (!/^kubernetes_/.test(t)) return null;
    if (/deployment/.test(t)) return "DEP";
    if (/stateful_set/.test(t)) return "STS";
    if (/daemon_set/.test(t)) return "DS";
    if (/cron_job/.test(t)) return "CJ";
    if (/_job/.test(t)) return "JOB";
    if (/network_policy/.test(t)) return "NP";
    return null;
  }

  function parseMap(src) {
    const out = {};
    const re = /("?)([\w./-]+)\1\s*=\s*"([^"]*)"/g;
    let m;
    while ((m = re.exec(src))) out[m[2]] = m[3];
    return out;
  }

  function blockMaps(raw, name) {
    const res = [];
    const re = new RegExp("\\b" + name + "\\s*=\\s*\\{([^}]*)\\}", "g");
    let m;
    while ((m = re.exec(raw))) res.push(parseMap(m[1]));
    return res;
  }

  function parseHCL(text) {
    const code = stripCode(text);
    const rawLines = text.split("\n");
    const lines = code.split("\n");
    const blocks = [];
    const errors = [];
    let depth = 0, cur = null, extraLine = 0;
    lines.forEach((ln, i) => {
      const raw = rawLines[i] || "";
      if (depth === 0) {
        const trimmed = ln.trim();
        if (trimmed) {
          let m = raw.match(/^\s*(resource|data)\s+"([^"]+)"\s+"([^"]+)"\s*\{/);
          let b = null;
          if (m) b = { kind: m[1], type: m[2], name: m[3] };
          else if ((m = raw.match(/^\s*(variable|output|module|provider)\s+"([^"]+)"\s*\{/))) b = { kind: m[1], type: m[1], name: m[2] };
          else if ((m = raw.match(/^\s*(terraform|locals|import|moved|removed)\s*\{/))) b = { kind: m[1], type: m[1], name: "" };
          if (b) { b.line = i + 1; b.start = i; cur = b; }
          else if (!extraLine) { extraLine = i + 1; errors.push({ line: i + 1, sev: "e", msg: "conteúdo fora de bloco: " + trunc(raw.trim(), 50) }); }
        }
      }
      for (const ch of ln) {
        if (ch === "{") depth++;
        else if (ch === "}") {
          depth--;
          if (depth < 0) { errors.push({ line: i + 1, sev: "e", msg: "} sobrando" }); depth = 0; }
          if (depth === 0 && cur) {
            cur.end = i;
            cur.raw = rawLines.slice(cur.start, i + 1).join("\n");
            cur.code = lines.slice(cur.start, i + 1).join("\n");
            blocks.push(cur);
            cur = null;
          }
        }
      }
    });
    if (depth > 0) errors.push({ line: cur ? cur.line : lines.length, sev: "e", msg: "{ sem fechamento" + (cur ? " no bloco " + (cur.type + " " + cur.name).trim() : "") });

    const res = {}, vars = [], outs = [], imports = [];
    const seen = {};
    blocks.forEach(b => {
      if (b.kind === "resource" || b.kind === "data") {
        const addr = (b.kind === "data" ? "data." : "") + b.type + "." + b.name;
        if (seen[addr]) errors.push({ line: b.line, sev: "e", msg: "recurso duplicado: " + addr });
        seen[addr] = true;
        b.addr = addr;
        res[addr] = b;
      } else if (b.kind === "variable") vars.push(b);
      else if (b.kind === "output") outs.push(b);
      else if (b.kind === "import") imports.push(b);
    });
    const varNames = new Set(vars.map(v => v.name));

    const lineOf = (b, idx) => b.line + (b.code.slice(0, idx).match(/\n/g) || []).length;
    const refsOf = b => {
      const refs = [];
      const re = /\b(data\.)?([a-z][a-z0-9]*_[a-z0-9_]+)\.([a-zA-Z_][\w-]*)/g;
      let m;
      while ((m = re.exec(b.code))) {
        if (BUILTIN.has(m[2])) continue;
        if (m.index > 0 && /[\w.]/.test(b.code[m.index - 1])) continue;
        refs.push({ addr: (m[1] || "") + m[2] + "." + m[3], idx: m.index });
      }
      return refs;
    };
    blocks.forEach(b => {
      const re = /\bvar\.([a-zA-Z_][\w-]*)/g;
      let m;
      while ((m = re.exec(b.code))) {
        if (!varNames.has(m[1])) errors.push({ line: lineOf(b, m.index), sev: "e", msg: "variável não declarada: var." + m[1] });
      }
      refsOf(b).forEach(r => {
        if (!res[r.addr]) errors.push({ line: lineOf(b, r.idx), sev: "w", msg: "referência a recurso não declarado: " + r.addr });
      });
    });

    const groupLabels = {}, groupTags = {};
    const nsAddrName = {};
    const vpcAddr = {};
    Object.keys(res).forEach(a => {
      const b = res[a];
      if (/^kubernetes_namespace/.test(b.type)) {
        const m = b.raw.match(/\bname\s*=\s*"([^"]+)"/);
        nsAddrName[a] = m ? m[1] : b.name;
      }
      if (b.type === "aws_vpc") vpcAddr[a] = true;
    });
    const members = {};
    const nodes = [], edges = [];
    Object.keys(res).forEach(a => {
      const b = res[a];
      let group = null;
      const nm = b.raw.match(/\bnamespace\s*=\s*(?:"([^"]+)"|((?:[a-z0-9_]+)\.[\w-]+)\.metadata)/);
      if (nm) {
        if (nm[1]) group = nm[1];
        else if (nsAddrName[nm[2]]) group = nsAddrName[nm[2]];
        if (group) { groupTags[group] = "namespace"; groupLabels[group] = group; }
      }
      const vm = b.code.match(/\bvpc_id\s*=\s*(aws_vpc\.[\w-]+)/);
      if (!group && vm && vpcAddr[vm[1]]) {
        group = "vpc:" + vm[1];
        groupTags[group] = "aws_vpc";
        groupLabels[group] = vm[1].split(".")[1];
      }
      b.group = group;
      if (group) members[group] = (members[group] || 0) + 1;
    });
    Object.keys(res).forEach(a => {
      const b = res[a];
      if (nsAddrName[a] && members[nsAddrName[a]]) return;
      if (vpcAddr[a] && members["vpc:" + a]) return;
      const kind = tfKind(b.type);
      const nameM = b.raw.match(/\bmetadata\s*\{[^}]*?\bname\s*=\s*"([^"]+)"/) || b.raw.match(/^\s*(?:name|bucket|identifier|cluster_name|replication_group_id)\s*=\s*"([^"]+)"/m);
      const label = nameM ? nameM[1] : b.name;
      const shortType = b.type.replace(/^kubernetes_/, "k8s_").replace(/_v\d$/, "");
      nodes.push({
        id: a, addr: a, kind: kind, name: label, sub: shortType + (label !== b.name ? " · " + b.name : ""),
        group: b.group, line: b.line, abbr: b.kind === "data" ? "DATA" : tfAbbr(b.type)
      });
    });
    const nodeIds = new Set(nodes.map(n => n.id));
    const resolve = a => {
      if (nodeIds.has(a)) return a;
      return null;
    };
    Object.keys(res).forEach(a => {
      const b = res[a];
      if (!nodeIds.has(a)) return;
      const dep = new Set();
      const dm = b.code.match(/\bdepends_on\s*=\s*\[([^\]]*)\]/);
      if (dm) dm[1].split(",").map(x => x.trim()).filter(Boolean).forEach(x => dep.add(x));
      const seenE = new Set();
      refsOf(b).forEach(r => {
        const to = resolve(r.addr);
        if (!to || to === a || seenE.has(to)) return;
        seenE.add(to);
        const isDep = dep.has(r.addr);
        let kind = isDep ? "depends" : "ref";
        const tk = tfKind(res[to].type);
        if (!isDep && (tk === "configmap" || tk === "secret")) kind = "config";
        if (!isDep && tk === "pvc") kind = "storage";
        if (!isDep && tfKind(b.type) === "hpa") kind = "scales";
        if (!isDep && (tk === "service" || tk === "workload" || tk === "db" || tk === "cache" || tk === "queue")) kind = tfKind(b.type) === "ingress" ? "traffic" : "routes";
        edges.push({ id: a + ">" + to, from: a, to: to, kind: kind });
      });
    });
    const svcs = Object.keys(res).filter(a => nodeIds.has(a) && tfKind(res[a].type) === "service");
    const wls = Object.keys(res).filter(a => nodeIds.has(a) && tfKind(res[a].type) === "workload");
    svcs.forEach(sa => {
      const sel = blockMaps(res[sa].raw, "selector")[0];
      if (!sel || !Object.keys(sel).length) return;
      wls.forEach(wa => {
        if (res[wa].group !== res[sa].group) return;
        const labelMaps = blockMaps(res[wa].raw, "labels").concat(blockMaps(res[wa].raw, "match_labels"));
        if (labelMaps.some(lm => Object.keys(sel).every(k => lm[k] === sel[k]))) edges.push({ id: sa + ">" + wa, from: sa, to: wa, kind: "routes" });
      });
    });
    imports.forEach(b => {
      const m = b.code.match(/\bto\s*=\s*((?:data\.)?[a-z0-9_]+\.[\w-]+)/);
      if (m && !res[m[1]]) errors.push({ line: b.line, sev: "w", msg: "import aponta para recurso inexistente: " + m[1] });
    });
    errors.sort((a, b) => a.line - b.line);
    return { nodes: nodes, edges: edges, errors: errors, vars: vars, outs: outs, imports: imports, resources: res, groupLabels: groupLabels, groupTags: groupTags };
  }

  function parseAndDraw(relayout) {
    const ta = document.getElementById("tf-ta");
    if (!ta || !S.tg) return;
    const p = parseHCL(ta.value);
    S.tf = p;
    const s = tfState();
    S.tg.setData(p.nodes, p.edges, s.pos, relayout);
    if (relayout) { s.pos = Object.assign({}, S.tg.st.pos); lsSet(tfKey(), s); }
    const empty = document.getElementById("tf-empty");
    if (empty) {
      if (!p.nodes.length) { empty.style.display = ""; empty.textContent = "Digite HCL à esquerda — ex.: resource \"aws_vpc\" \"main\" { cidr_block = \"10.0.0.0/16\" }"; }
      else empty.style.display = "none";
    }
    const errs = p.errors.filter(e => e.sev === "e").length;
    document.getElementById("tf-errc").textContent = p.errors.length ? p.errors.length : "";
    document.getElementById("tf-cnt").innerHTML = '<span style="color:var(--grn)">' + Object.keys(p.resources).length + "</span> recursos" + (errs ? ' · <span style="color:var(--red)">' + errs + " erro" + (errs > 1 ? "s" : "") + "</span>" : "");
    paintEditor();
    renderBottom();
    runSearch(false);
  }

  let bottomTab = "problems";
  function renderBottom() {
    const p = S.tf;
    const tabs = document.getElementById("tf-btabs"), panel = document.getElementById("tf-bpanel");
    if (!p || !tabs) return;
    const defs = [
      ["problems", "Problemas", p.errors.length],
      ["vars", "Variáveis", p.vars.length],
      ["outs", "Outputs", p.outs.length],
      ["res", "Recursos", Object.keys(p.resources).length],
      ["imports", "Imports", p.imports.length]
    ];
    tabs.innerHTML = defs.map(d => '<span class="tf-btab' + (d[0] === bottomTab ? " on" : "") + '" data-bt="' + d[0] + '">' + d[1] +
      '<span class="n' + (d[0] === "problems" && d[2] ? "" : " z") + '">' + d[2] + "</span></span>").join("");
    tabs.querySelectorAll("[data-bt]").forEach(x => x.onclick = () => { bottomTab = x.getAttribute("data-bt"); renderBottom(); });
    let rows = [];
    if (bottomTab === "problems") rows = p.errors.map(e => [e.line, '<span class="' + e.sev + '">' + (e.sev === "e" ? "erro" : "aviso") + "</span> · linha " + e.line + " — " + h(e.msg)]);
    if (bottomTab === "vars") rows = p.vars.map(v => [v.line, '<span class="hl-v">var.' + h(v.name) + "</span>"]);
    if (bottomTab === "outs") rows = p.outs.map(v => [v.line, h(v.name)]);
    if (bottomTab === "res") rows = Object.keys(p.resources).map(a => [p.resources[a].line, '<span class="hl-r">' + h(a) + "</span>"]);
    if (bottomTab === "imports") rows = p.imports.map(b => { const m = b.raw.match(/\bid\s*=\s*"([^"]*)"/); const t = b.raw.match(/\bto\s*=\s*([\w.-]+)/); return [b.line, h(t ? t[1] : "?") + ' <span class="muted">← ' + h(m ? m[1] : "") + "</span>"]; });
    panel.innerHTML = rows.length ? rows.map(r => '<div data-line="' + r[0] + '">' + r[1] + "</div>").join("") : '<span class="muted">' + (bottomTab === "problems" ? "nenhum problema encontrado" : "vazio") + "</span>";
    panel.querySelectorAll("[data-line]").forEach(x => x.onclick = () => gotoLine(+x.getAttribute("data-line"), true));
  }

  const TEMPLATES = [
    ["Kubernetes", [
      ["kubernetes_deployment_v1", "app", n => 'resource "kubernetes_deployment_v1" "' + n + '" {\n  metadata {\n    name      = "' + n + '"\n    namespace = "default"\n    labels = {\n      app = "' + n + '"\n    }\n  }\n  spec {\n    replicas = 2\n    selector {\n      match_labels = {\n        app = "' + n + '"\n      }\n    }\n    template {\n      metadata {\n        labels = {\n          app = "' + n + '"\n        }\n      }\n      spec {\n        container {\n          name  = "' + n + '"\n          image = "nginx:1.27"\n          port {\n            container_port = 80\n          }\n        }\n      }\n    }\n  }\n}\n'],
      ["kubernetes_service_v1", "app", n => 'resource "kubernetes_service_v1" "' + n + '" {\n  metadata {\n    name      = "' + n + '"\n    namespace = "default"\n  }\n  spec {\n    selector = {\n      app = "' + n + '"\n    }\n    port {\n      port        = 80\n      target_port = "80"\n    }\n  }\n}\n'],
      ["kubernetes_ingress_v1", "web", n => 'resource "kubernetes_ingress_v1" "' + n + '" {\n  metadata {\n    name      = "' + n + '"\n    namespace = "default"\n  }\n  spec {\n    ingress_class_name = "alb"\n    rule {\n      host = "' + n + '.example.com"\n      http {\n        path {\n          path      = "/"\n          path_type = "Prefix"\n          backend {\n            service {\n              name = "app"\n              port {\n                number = 80\n              }\n            }\n          }\n        }\n      }\n    }\n  }\n}\n'],
      ["kubernetes_config_map_v1", "config", n => 'resource "kubernetes_config_map_v1" "' + n + '" {\n  metadata {\n    name      = "' + n + '"\n    namespace = "default"\n  }\n  data = {\n    LOG_LEVEL = "info"\n  }\n}\n'],
      ["kubernetes_horizontal_pod_autoscaler_v2", "app", n => 'resource "kubernetes_horizontal_pod_autoscaler_v2" "' + n + '" {\n  metadata {\n    name      = "' + n + '"\n    namespace = "default"\n  }\n  spec {\n    min_replicas = 2\n    max_replicas = 10\n    scale_target_ref {\n      api_version = "apps/v1"\n      kind        = "Deployment"\n      name        = "' + n + '"\n    }\n  }\n}\n'],
      ["kubernetes_namespace_v1", "apps", n => 'resource "kubernetes_namespace_v1" "' + n + '" {\n  metadata {\n    name = "' + n + '"\n  }\n}\n']
    ]],
    ["AWS", [
      ["aws_vpc", "main", n => 'resource "aws_vpc" "' + n + '" {\n  cidr_block = "10.0.0.0/16"\n}\n'],
      ["aws_subnet", "private_a", n => 'resource "aws_subnet" "' + n + '" {\n  vpc_id     = aws_vpc.main.id\n  cidr_block = "10.0.1.0/24"\n}\n'],
      ["aws_security_group", "app", n => 'resource "aws_security_group" "' + n + '" {\n  name   = "' + n + '"\n  vpc_id = aws_vpc.main.id\n}\n'],
      ["aws_db_instance", "postgres", n => 'resource "aws_db_instance" "' + n + '" {\n  identifier        = "' + n.replace(/_/g, "-") + '"\n  engine            = "postgres"\n  instance_class    = "db.t4g.micro"\n  allocated_storage = 20\n}\n'],
      ["aws_elasticache_replication_group", "redis", n => 'resource "aws_elasticache_replication_group" "' + n + '" {\n  replication_group_id = "' + n.replace(/_/g, "-") + '"\n  description          = "' + n + '"\n  engine               = "redis"\n  node_type            = "cache.t4g.micro"\n}\n'],
      ["aws_s3_bucket", "assets", n => 'resource "aws_s3_bucket" "' + n + '" {\n  bucket = "' + n.replace(/_/g, "-") + '"\n}\n'],
      ["aws_sqs_queue", "jobs", n => 'resource "aws_sqs_queue" "' + n + '" {\n  name = "' + n + '"\n}\n'],
      ["aws_lb", "public", n => 'resource "aws_lb" "' + n + '" {\n  name               = "' + n + '"\n  load_balancer_type = "application"\n}\n'],
      ["aws_eks_cluster", "main", n => 'resource "aws_eks_cluster" "' + n + '" {\n  name     = "' + n + '"\n  role_arn = "arn:aws:iam::123456789012:role/eks"\n  vpc_config {\n    subnet_ids = [aws_subnet.private_a.id]\n  }\n}\n']
    ]]
  ];

  function openTFMenu(w, sx, sy) {
    closePop();
    const m = document.createElement("div");
    m.className = "tf-menu";
    m.id = "tf-menu";
    m.style.left = Math.max(8, sx) + "px";
    m.style.top = Math.max(8, sy) + "px";
    m.innerHTML = TEMPLATES.map(g => '<div class="h">' + g[0] + "</div>" + g[1].map((t, i) => '<div data-t="' + g[0] + ":" + i + '">' + t[0] + "</div>").join("")).join("");
    const host = document.getElementById("tf-stage");
    host.appendChild(m);
    m.querySelectorAll("[data-t]").forEach(x => x.onclick = () => {
      const [g, i] = x.getAttribute("data-t").split(":");
      const t = TEMPLATES.find(z => z[0] === g)[1][+i];
      const ta = document.getElementById("tf-ta");
      const used = S.tf ? S.tf.resources : {};
      let name = t[1], k = 2;
      while (used[t[0] + "." + name]) name = t[1] + "_" + k++;
      const addr = t[0] + "." + name;
      const s = tfState();
      s.pos = s.pos || {};
      s.pos[addr] = { x: Math.round(w.x - W / 2), y: Math.round(w.y - H / 2) };
      lsSet(tfKey(), s);
      ta.value = ta.value.replace(/\s*$/, "") + "\n\n" + t[2](name);
      ta.dispatchEvent(new Event("input"));
      clearTimeout(S.tfParseT);
      parseAndDraw(false);
      closePop();
      const n = S.tf.nodes.find(z => z.addr === addr);
      if (n) gotoLine(n.line, true);
    });
    setTimeout(() => document.addEventListener("pointerdown", function off(ev) {
      if (!m.contains(ev.target)) { closePop(); document.removeEventListener("pointerdown", off); }
    }), 0);
  }

  function tfAddDepends(from, to) {
    if (!S.tf) return;
    const b = S.tf.resources[from];
    if (!b || !S.tf.resources[to]) return;
    const ta = document.getElementById("tf-ta");
    const lines = ta.value.split("\n");
    const blockLines = lines.slice(b.start, b.end + 1).join("\n");
    let newBlock;
    const dm = blockLines.match(/(\bdepends_on\s*=\s*\[)([^\]]*)\]/);
    if (dm) {
      if (dm[2].split(",").map(x => x.trim()).indexOf(to) >= 0) return;
      const inner = dm[2].trim() ? dm[2].replace(/\s*$/, "") + ", " + to : to;
      newBlock = blockLines.replace(dm[0], dm[1] + inner + "]");
    } else {
      const bl = blockLines.split("\n");
      bl.splice(bl.length - 1, 0, "", "  depends_on = [" + to + "]");
      newBlock = bl.join("\n");
    }
    lines.splice(b.start, b.end - b.start + 1, newBlock);
    ta.value = lines.join("\n");
    ta.dispatchEvent(new Event("input"));
    clearTimeout(S.tfParseT);
    parseAndDraw(false);
    gotoLine(b.line);
  }
})();
