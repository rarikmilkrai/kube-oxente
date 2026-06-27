package main

const indexHTML = `<!doctype html><html lang="pt-br"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>kdash</title>
<style>
:root{
 --bg:#0e1016;--panel:#161922;--panel2:#1d212c;--bd:#262b38;--fg:#e7e9ef;--mut:#8a93a6;
 --red:#ff6b6b;--yel:#ffc04d;--grn:#5fd38a;--blu:#5aa9ff;--cmd:#8ac28e;
 --prod:#ff6b6b;--staging:#ffc04d;--dev:#5aa9ff;--other:#7a8194;
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
 font:13.5px/1.5 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}
code,.mono{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
a{color:var(--blu)}
.app{display:grid;grid-template-columns:200px 1fr;grid-template-rows:52px 1fr;height:100vh}
header{grid-column:1/3;display:flex;align-items:center;gap:14px;padding:0 16px;
 border-bottom:1px solid var(--bd);background:var(--panel)}
.brand{font-weight:700;color:var(--blu);font-size:15px;letter-spacing:.3px}
select{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);border-radius:6px;
 padding:5px 8px;font:inherit;max-width:340px}
.badge{padding:2px 9px;border-radius:20px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.5px}
.badge.prod{background:rgba(255,107,107,.16);color:var(--prod)}
.badge.staging{background:rgba(255,192,77,.16);color:var(--staging)}
.badge.dev{background:rgba(90,169,255,.16);color:var(--dev)}
.badge.other{background:rgba(122,129,148,.16);color:var(--other)}
.right{margin-left:auto;display:flex;align-items:center;gap:12px;color:var(--mut);font-size:12px}
.tgl{cursor:pointer;user-select:none}
button{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);border-radius:6px;
 padding:5px 11px;cursor:pointer;font:inherit}
button:hover{background:#262b38}
nav{border-right:1px solid var(--bd);background:var(--panel);padding:8px 0;overflow:auto}
nav a{display:flex;align-items:center;gap:10px;padding:9px 16px;color:var(--mut);
 cursor:pointer;text-decoration:none;border-left:3px solid transparent}
nav a:hover{color:var(--fg);background:var(--panel2)}
nav a.on{color:var(--fg);border-left-color:var(--blu);background:var(--panel2)}
nav .ico{width:16px;text-align:center;opacity:.85}
nav .cnt{margin-left:auto;font-size:11px;color:var(--mut)}
nav a.on .cnt,.cnt.red{color:var(--red)}
main{overflow:auto;padding:18px 22px}
h2{margin:0 0 14px;font-size:18px}
.spin{color:var(--mut);padding:40px;text-align:center}
.cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:12px;margin-bottom:18px}
.card{background:var(--panel);border:1px solid var(--bd);border-radius:10px;padding:14px 16px}
.card .n{font-size:26px;font-weight:700}
.card .l{color:var(--mut);font-size:12px;margin-top:2px}
.card.red .n{color:var(--red)}.card.yel .n{color:var(--yel)}.card.grn .n{color:var(--grn)}
.sec{background:var(--panel);border:1px solid var(--bd);border-radius:10px;padding:14px 16px;margin-bottom:16px}
.sec h3{margin:0 0 10px;font-size:12px;color:var(--mut);text-transform:uppercase;letter-spacing:.6px}
.bar{display:flex;height:10px;border-radius:6px;overflow:hidden;background:#0b0d12}
.bar i{display:block;height:100%}
.legend{display:flex;gap:16px;margin-top:8px;color:var(--mut);font-size:12px;flex-wrap:wrap}
.dotc{display:inline-block;width:9px;height:9px;border-radius:50%;margin-right:5px;vertical-align:baseline}
.wl{display:flex;justify-content:space-between;padding:6px 0;border-bottom:1px solid var(--bd)}
.wl:last-child{border-bottom:0}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;color:var(--mut);font-weight:600;font-size:11px;text-transform:uppercase;
 letter-spacing:.5px;padding:8px 10px;border-bottom:1px solid var(--bd);position:sticky;top:0;background:var(--bg)}
td{padding:8px 10px;border-bottom:1px solid var(--bd);vertical-align:top}
tr.clk{cursor:pointer}tr.clk:hover td{background:var(--panel2)}
.dot{font-size:10px}.s2{color:var(--red)}.s1{color:var(--yel)}.s0{color:var(--grn)}
.muted{color:var(--mut)}
.filters{display:flex;gap:10px;margin-bottom:12px;flex-wrap:wrap;align-items:center}
input[type=text]{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);
 border-radius:6px;padding:6px 10px;font:inherit;min-width:200px}
.diagln{color:var(--blu);font-size:12px}
.pill{padding:1px 7px;border-radius:5px;font-size:11px;background:var(--panel2);color:var(--mut)}
.ok{color:var(--grn)}.bad{color:var(--red)}
/* modal */
.ov{position:fixed;inset:0;background:rgba(0,0,0,.55);display:none;z-index:20}
.ov.show{display:block}
.modal{position:fixed;top:0;right:0;height:100vh;width:min(720px,92vw);background:var(--panel);
 border-left:1px solid var(--bd);z-index:21;transform:translateX(100%);transition:transform .18s;
 display:flex;flex-direction:column}
.modal.show{transform:translateX(0)}
.mh{padding:14px 18px;border-bottom:1px solid var(--bd);display:flex;align-items:center;gap:10px}
.mh .x{margin-left:auto;cursor:pointer;color:var(--mut);font-size:20px}
.mb{padding:16px 18px;overflow:auto}
.ev{font-size:12px;color:var(--mut);margin:3px 0}.ev.w{color:var(--yel)}
.diag{color:var(--blu);margin:6px 0}
.cmd{color:var(--cmd);font-size:12px;white-space:pre-wrap;margin-left:14px}
.mono.log{background:#0b0d12;border-radius:8px;padding:12px;font-size:12px;white-space:pre-wrap;
 word-break:break-all;max-height:48vh;overflow:auto}
.log .e{color:var(--red)}.log .nn{color:var(--mut)}
.empty{color:var(--grn);text-align:center;padding:40px}
</style></head><body>
<div class="app">
 <header>
  <span class="brand">koxente</span>
  <select id="ctxsel" onchange="switchCtx()"></select>
  <span class="badge" id="envb"></span>
  <span class="right">
   <span id="gen"></span>
   <span class="tgl" id="autob" onclick="toggleAuto()">&#x21bb; auto: on</span>
   <button onclick="refresh()">atualizar</button>
  </span>
 </header>
 <nav id="nav"></nav>
 <main id="main"><div class="spin">carregando&#8230;</div></main>
</div>
<div class="ov" id="ov" onclick="closeDetail()"></div>
<div class="modal" id="modal"><div class="mh" id="mh"></div><div class="mb" id="mb"></div></div>
<script>
var VIEWS=[
 {id:"overview",ico:"&#9632;",label:"Overview"},
 {id:"problems",ico:"&#9888;",label:"Problemas"},
 {id:"pods",ico:"&#9673;",label:"Pods"},
 {id:"workloads",ico:"&#9707;",label:"Workloads"},
 {id:"services",ico:"&#8644;",label:"Services"},
 {id:"namespaces",ico:"&#9783;",label:"Namespaces"},
 {id:"events",ico:"&#9993;",label:"Eventos"},
 {id:"nodes",ico:"&#9636;",label:"N&oacute;s"}
];
var view="overview", auto=true, timer=null, counts={}, podsCache=[], filt={env:"",q:""};
function esc(s){return (s==null?"":""+s).replace(/[&<>"]/g,function(c){return({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}[c])})}
function api(p){return fetch(p).then(function(r){return r.json()})}
function envBadge(e){return '<span class="badge '+(e||"other")+'">'+esc(e||"other")+'</span>'}
function sev(s){return '<span class="dot s'+s+'">&#9679;</span>'}

function renderNav(){
 document.getElementById("nav").innerHTML=VIEWS.map(function(v){
  var c=counts[v.id]!=null?'<span class="cnt'+(v.id=="problems"&&counts[v.id]>0?" red":"")+'">'+counts[v.id]+'</span>':"";
  return '<a class="'+(v.id==view?"on":"")+'" onclick="nav(\''+v.id+'\')"><span class="ico">'+v.ico+'</span>'+v.label+c+'</a>';
 }).join("");
}
function nav(v){view=v;renderNav();render()}

function loadContexts(){
 api("/api/contexts").then(function(d){
  var s=document.getElementById("ctxsel");
  s.innerHTML=(d.all||[]).map(function(c){return '<option'+(c==d.active?" selected":"")+'>'+esc(c)+'</option>'}).join("")
   ||'<option>'+esc(d.active||"(sem contexto)")+'</option>';
 });
}
function switchCtx(){
 var c=document.getElementById("ctxsel").value;
 api("/api/setcontext?name="+encodeURIComponent(c)).then(function(){counts={};refresh()});
}
function toggleAuto(){auto=!auto;document.getElementById("autob").innerHTML="&#x21bb; auto: "+(auto?"on":"off");schedule()}
function schedule(){clearTimeout(timer);if(auto)timer=setTimeout(refresh,7000)}
function refresh(){render();schedule()}

function render(){
 renderNav();
 var m=document.getElementById("main");
 var fns={overview:rOverview,problems:rProblems,pods:rPods,workloads:rWorkloads,
  services:rServices,namespaces:rNamespaces,events:rEvents,nodes:rNodes};
 (fns[view]||rOverview)(m);
}

function setGen(){document.getElementById("gen").textContent="atualizado "+new Date().toLocaleTimeString()}

function rOverview(m){
 api("/api/overview").then(function(d){
  setGen();
  var eb=document.getElementById("envb");if(eb){eb.className="badge "+(d.env||"other");eb.textContent=(d.env||"other")}
  var p=d.pods||{};
  var html='<h2>Overview <span class="muted" style="font-size:13px">'+esc(d.ctx)+'</span></h2>';
  if(d.err){m.innerHTML=html+'<div class="spin">erro: '+esc(d.err)+'</div>';return}
  html+='<div class="cards">'+
   card(p.Total,"pods")+
   card(p.Running,"running","grn")+
   card(p.Pending,"pending",p.Pending?"yel":"")+
   card(p.Failed+p.Prob,"com falha",(p.Failed+p.Prob)?"red":"")+
   card(d.nsCount,"namespaces")+
   card(d.nodesTotal,"n&oacute;s"+(d.nodesDown?" ("+d.nodesDown+" down)":""),d.nodesDown?"red":"")+
   card(d.probApps,"apps c/ falha",d.probApps?"red":"")+
   '</div>';
  // saude
  var tot=p.OK+p.Warn+p.Prob||1;
  html+='<div class="sec"><h3>Sa&uacute;de dos pods</h3><div class="bar">'+
   barseg(p.OK,tot,"var(--grn)")+barseg(p.Warn,tot,"var(--yel)")+barseg(p.Prob,tot,"var(--red)")+
   '</div><div class="legend">'+
   leg("var(--grn)","ok "+p.OK)+leg("var(--yel)","aten&ccedil;&atilde;o "+p.Warn)+leg("var(--red)","problema "+p.Prob)+
   '</div></div>';
  // ambiente
  if(d.envSplit&&d.envSplit.length){
   html+='<div class="sec"><h3>Pods por ambiente</h3>';
   d.envSplit.forEach(function(e){html+='<div class="wl"><span>'+envBadge(e.label)+'</span><b>'+e.value+'</b></div>'});
   html+='</div>';
  }
  // workloads
  html+='<div class="sec"><h3>Workloads prontos</h3>';
  (d.workloads||[]).forEach(function(w){
   var ok=w.ready>=w.total;
   html+='<div class="wl"><span>'+esc(w.kind)+'s</span><span class="'+(ok?"ok":"bad")+'">'+w.ready+'/'+w.total+' prontos</span></div>';
  });
  html+='</div>';
  // nos
  html+='<div class="sec"><h3>N&oacute;s</h3><div class="muted mono">'+(d.nodes||[]).map(esc).join(" &middot; ")+'</div></div>';
  m.innerHTML=html;
 });
}
function card(n,l,cls){return '<div class="card '+(cls||"")+'"><div class="n">'+n+'</div><div class="l">'+l+'</div></div>'}
function barseg(v,tot,c){var w=(v/tot*100).toFixed(1);return '<i style="width:'+w+'%;background:'+c+'"></i>'}
function leg(c,t){return '<span><span class="dotc" style="background:'+c+'"></span>'+t+'</span>'}

function rProblems(m){
 api("/api/problems").then(function(d){
  setGen();counts.problems=d.length;renderNav();
  if(!d.length){m.innerHTML='<h2>Problemas</h2><div class="empty">&#10003; nenhuma aplica&ccedil;&atilde;o com falha</div>';return}
  var rows=d.map(function(a){
   var grp=a.podCount>1?' <span class="pill">'+a.podCount+' pods</span>':"";
   var last=a.lastExit?' &middot; &uacute;ltimo restart h&aacute; '+esc(a.lastExit):"";
   return '<tr class="clk" onclick="openDetail(\''+esc(a.ns)+'\',\''+esc(a.samplePod)+'\',\''+esc(a.sampleReason)+'\')">'+
    '<td>'+sev(a.severity)+'</td>'+
    '<td><b>'+esc(a.app)+'</b>'+grp+'<div class="muted" style="font-size:12px">'+esc(a.ns)+'</div></td>'+
    '<td>'+envBadge(a.env)+'</td>'+
    '<td class="s'+a.severity+'">'+esc(a.reason)+'<div class="muted" style="font-size:11px">restarts '+a.restarts+last+'</div></td>'+
    '<td class="diagln">'+esc(a.primaryDiag)+'</td></tr>';
  }).join("");
  m.innerHTML='<h2>Problemas <span class="muted" style="font-size:13px">'+d.length+' apps</span></h2>'+
   '<table><thead><tr><th></th><th>Aplica&ccedil;&atilde;o</th><th>Amb.</th><th>Motivo</th><th>Diagn&oacute;stico prov&aacute;vel</th></tr></thead><tbody>'+rows+'</tbody></table>';
 });
}

function rPods(m){
 api("/api/pods").then(function(d){
  setGen();podsCache=d;
  m.innerHTML='<h2>Pods</h2>'+
   '<div class="filters">'+
   '<select id="fenv" onchange="filt.env=this.value;drawPods()"><option value="">todos ambientes</option><option>prod</option><option>staging</option><option>dev</option><option>other</option></select>'+
   '<select id="fst" onchange="filt.st=this.value;drawPods()"><option value="">todos status</option><option value="2">problema</option><option value="1">aten&ccedil;&atilde;o</option><option value="0">ok</option></select>'+
   '<input type="text" id="fq" placeholder="filtrar por nome/namespace&#8230;" oninput="filt.q=this.value;drawPods()">'+
   '<span class="muted" id="pcount"></span></div>'+
   '<table><thead><tr><th></th><th>Namespace</th><th>Pod</th><th>Amb.</th><th>Status</th><th>Ready</th><th>Restarts</th><th>Idade</th><th>N&oacute;</th></tr></thead><tbody id="pbody"></tbody></table>';
  drawPods();
 });
}
function drawPods(){
 var d=podsCache.filter(function(p){
  if(filt.env&&p.env!=filt.env)return false;
  if(filt.st!==""&&filt.st!=null&&(""+p.severity)!=filt.st)return false;
  if(filt.q){var q=filt.q.toLowerCase();if((p.name+" "+p.ns).toLowerCase().indexOf(q)<0)return false}
  return true;
 });
 document.getElementById("pcount").textContent=d.length+" de "+podsCache.length;
 document.getElementById("pbody").innerHTML=d.map(function(p){
  return '<tr class="clk" onclick="openDetail(\''+esc(p.ns)+'\',\''+esc(p.name)+'\',\''+esc(p.reason)+'\')">'+
   '<td>'+sev(p.severity)+'</td><td class="muted">'+esc(p.ns)+'</td><td>'+esc(p.name)+'</td>'+
   '<td>'+envBadge(p.env)+'</td><td class="s'+p.severity+'">'+esc(p.reason)+'</td>'+
   '<td>'+p.ready+'/'+p.total+'</td><td>'+p.restarts+'</td><td>'+esc(p.age)+'</td><td class="muted mono">'+esc(p.node)+'</td></tr>';
 }).join("")||'<tr><td colspan="9" class="empty">nenhum pod</td></tr>';
}

function rWorkloads(m){
 api("/api/workloads").then(function(d){
  setGen();
  m.innerHTML='<h2>Workloads</h2><table><thead><tr><th>Tipo</th><th>Namespace</th><th>Nome</th><th>Amb.</th><th>Ready</th></tr></thead><tbody>'+
   d.map(function(w){return '<tr><td><span class="pill">'+esc(w.kind)+'</span></td><td class="muted">'+esc(w.ns)+'</td><td>'+esc(w.name)+'</td>'+
    '<td>'+envBadge(w.env)+'</td><td class="'+(w.ok?"ok":"bad")+'">'+w.ready+'/'+w.desired+'</td></tr>'}).join("")+
   '</tbody></table>';
 });
}
function rServices(m){
 api("/api/services").then(function(d){
  setGen();
  m.innerHTML='<h2>Services</h2><table><thead><tr><th>Namespace</th><th>Nome</th><th>Tipo</th><th>ClusterIP</th><th>Portas</th><th>Amb.</th></tr></thead><tbody>'+
   d.map(function(s){return '<tr><td class="muted">'+esc(s.ns)+'</td><td>'+esc(s.name)+'</td><td>'+esc(s.type)+'</td>'+
    '<td class="mono muted">'+esc(s.ip)+'</td><td class="mono">'+esc(s.ports)+'</td><td>'+envBadge(s.env)+'</td></tr>'}).join("")+
   '</tbody></table>';
 });
}
function rNamespaces(m){
 api("/api/namespaces").then(function(d){
  setGen();counts.namespaces=d.length;renderNav();
  m.innerHTML='<h2>Namespaces</h2><table><thead><tr><th>Namespace</th><th>Amb.</th><th>Pods</th><th>OK</th><th>Aten&ccedil;&atilde;o</th><th>Problema</th></tr></thead><tbody>'+
   d.map(function(n){return '<tr><td>'+esc(n.namespace)+'</td><td>'+envBadge(n.env)+'</td><td>'+n.pods+'</td>'+
    '<td class="ok">'+n.running+'</td><td class="s1">'+(n.warn||"")+'</td><td class="s2">'+(n.prob||"")+'</td></tr>'}).join("")+
   '</tbody></table>';
 });
}
function rEvents(m){
 api("/api/events").then(function(d){
  setGen();counts.events=d.length;renderNav();
  if(!d.length){m.innerHTML='<h2>Eventos</h2><div class="empty">&#10003; sem eventos de Warning</div>';return}
  m.innerHTML='<h2>Eventos <span class="muted" style="font-size:13px">Warning</span></h2><table><thead><tr><th>Namespace</th><th>Motivo</th><th>Objeto</th><th>Mensagem</th><th>Qtd</th></tr></thead><tbody>'+
   d.map(function(e){return '<tr><td class="muted">'+esc(e.ns)+'</td><td class="s1">'+esc(e.reason)+'</td><td class="mono" style="font-size:12px">'+esc(e.object)+'</td>'+
    '<td class="muted" style="font-size:12px">'+esc(e.message)+'</td><td>'+e.count+'</td></tr>'}).join("")+
   '</tbody></table>';
 });
}
function rNodes(m){
 api("/api/nodes").then(function(d){
  setGen();counts.nodes=d.length;renderNav();
  m.innerHTML='<h2>N&oacute;s</h2><table><thead><tr><th></th><th>Nome</th><th>Tipo</th><th>Capacidade</th><th>Zona</th><th>Idade</th></tr></thead><tbody>'+
   d.map(function(n){return '<tr><td><span class="dot '+(n.ready?"s0":"s2")+'">&#9679;</span></td><td class="mono">'+esc(n.name)+'</td>'+
    '<td>'+esc(n.type)+'</td><td>'+esc(n.cap||"-")+'</td><td class="muted">'+esc(n.zone)+'</td><td>'+esc(n.age)+'</td></tr>'}).join("")+
   '</tbody></table>';
 });
}

function openDetail(ns,name,reason){
 document.getElementById("ov").classList.add("show");
 var md=document.getElementById("modal");md.classList.add("show");
 document.getElementById("mh").innerHTML='<div><b>'+esc(name)+'</b><div class="muted" style="font-size:12px">'+esc(ns)+' &middot; '+esc(reason)+'</div></div><span class="x" onclick="closeDetail()">&times;</span>';
 document.getElementById("mb").innerHTML='<div class="spin">carregando&#8230;</div>';
 api("/api/detail?ns="+encodeURIComponent(ns)+"&name="+encodeURIComponent(name)+"&reason="+encodeURIComponent(reason)).then(function(d){
  var h="";
  if(d.diags&&d.diags.length){h+='<h3 class="muted">Diagn&oacute;stico prov&aacute;vel</h3>';
   d.diags.forEach(function(g){h+='<div class="diag">&rarr; '+esc(g.title)+'</div>';
    (g.cmds||[]).forEach(function(c){h+='<div class="cmd mono">'+esc(c)+'</div>'})});}
  if(d.events&&d.events.length){h+='<h3 class="muted" style="margin-top:14px">Eventos</h3>';
   d.events.forEach(function(e){h+='<div class="ev'+(e.type=="Warning"?" w":"")+'">&#9656; '+esc(e.reason)+': '+esc(e.message)+' <span style="opacity:.6">('+esc(e.age)+')</span></div>'})}
  h+='<h3 class="muted" style="margin-top:14px">Logs</h3><div class="mono log">'+
   ((d.logs&&d.logs.length)?d.logs.map(function(l){return '<span class="'+(l.isErr?"e":"nn")+'">'+esc(l.text)+'</span>'}).join("\n"):'<span class="nn">(sem logs)</span>')+'</div>';
  document.getElementById("mb").innerHTML=h;
 });
}
function closeDetail(){document.getElementById("ov").classList.remove("show");document.getElementById("modal").classList.remove("show")}
document.addEventListener("keydown",function(e){if(e.key=="Escape")closeDetail()});

loadContexts();render();schedule();
</script></body></html>`
