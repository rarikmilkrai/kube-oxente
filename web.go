package main

const indexHTML = `<!doctype html><html lang="pt-br"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Kube Oxente</title>
<link rel="stylesheet" href="/static/arch.css">
<script src="/static/arch.js"></script>
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
header{grid-column:1/3;display:flex;align-items:center;gap:10px;padding:0 14px;
 border-bottom:1px solid var(--bd);background:var(--panel);overflow:hidden}
.brand{font-weight:700;color:var(--blu);font-size:15px;letter-spacing:.3px;white-space:nowrap}
select{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);border-radius:6px;
 padding:5px 8px;font:inherit}
#ctxsel{max-width:280px}
#nsGlobal{max-width:170px;color:var(--mut);font-size:12px}
.badge{padding:2px 9px;border-radius:20px;font-size:11px;font-weight:600;text-transform:uppercase;
 letter-spacing:.5px;white-space:nowrap}
.badge.prod{background:rgba(255,107,107,.16);color:var(--prod)}
.badge.staging{background:rgba(255,192,77,.16);color:var(--staging)}
.badge.dev{background:rgba(90,169,255,.16);color:var(--dev)}
.badge.other{background:rgba(122,129,148,.16);color:var(--other)}
.right{margin-left:auto;display:flex;align-items:center;gap:8px;color:var(--mut);
 font-size:12px;white-space:nowrap;flex-shrink:0}
.tgl{cursor:pointer;user-select:none}
button{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);border-radius:6px;
 padding:5px 11px;cursor:pointer;font:inherit}
button:hover{background:#262b38}
.btn-live{background:rgba(95,211,138,.12);color:var(--grn);border-color:rgba(95,211,138,.3)}
.btn-live:hover{background:rgba(95,211,138,.2)}
.btn-stop{background:rgba(255,107,107,.12);color:var(--red);border-color:rgba(255,107,107,.3)}
nav{border-right:1px solid var(--bd);background:var(--panel);padding:8px 0;overflow-y:auto}
nav a{display:flex;align-items:center;gap:10px;padding:8px 16px;color:var(--mut);
 cursor:pointer;text-decoration:none;border-left:3px solid transparent;font-size:13px}
nav a:hover{color:var(--fg);background:var(--panel2)}
nav a.on{color:var(--fg);border-left-color:var(--blu);background:var(--panel2)}
nav .ico{width:16px;text-align:center;opacity:.85;flex-shrink:0}
nav .cnt{margin-left:auto;font-size:11px;color:var(--mut)}
nav a.on .cnt,.cnt.red{color:var(--red)}
main{overflow:auto;padding:18px 22px}
h2{margin:0 0 14px;font-size:18px}
.spin{color:var(--mut);padding:40px;text-align:center}
.cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(145px,1fr));gap:12px;margin-bottom:18px}
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
td{padding:7px 10px;border-bottom:1px solid var(--bd);vertical-align:middle}
tr.clk{cursor:pointer}tr.clk:hover td{background:var(--panel2)}
.dot{font-size:10px}.s2{color:var(--red)}.s1{color:var(--yel)}.s0{color:var(--grn)}
.muted{color:var(--mut)}
.filters{display:flex;gap:10px;margin-bottom:12px;flex-wrap:wrap;align-items:center}
input[type=text]{background:var(--panel2);color:var(--fg);border:1px solid var(--bd);
 border-radius:6px;padding:6px 10px;font:inherit;min-width:200px}
.diagln{color:var(--blu);font-size:12px}
.pill{padding:1px 7px;border-radius:5px;font-size:11px;background:var(--panel2);color:var(--mut)}
.ok{color:var(--grn)}.bad{color:var(--red)}
.met{font-size:11px;font-family:ui-monospace,Consolas,monospace;color:var(--mut)}
.met.hi{color:var(--yel)}.met.crit{color:var(--red)}
.spark{display:inline-block;vertical-align:middle;margin-left:3px;opacity:.75}
.qbar{height:5px;border-radius:3px;background:#0b0d12;width:80px;overflow:hidden;display:inline-block;vertical-align:middle;margin-left:6px}
.qbar i{display:block;height:100%}
/* modal */
.ov{position:fixed;inset:0;background:rgba(0,0,0,.55);display:none;z-index:20}
.ov.show{display:block}
.modal{position:fixed;top:0;right:0;height:100vh;width:min(740px,93vw);background:var(--panel);
 border-left:1px solid var(--bd);z-index:21;transform:translateX(100%);transition:transform .18s;
 display:flex;flex-direction:column}
.modal.show{transform:translateX(0)}
.mh{padding:14px 18px;border-bottom:1px solid var(--bd);display:flex;align-items:center;gap:10px;flex-shrink:0}
.mh .x{margin-left:auto;cursor:pointer;color:var(--mut);font-size:20px}
.mb{padding:16px 18px;overflow:auto;flex:1}
.ev{font-size:12px;color:var(--mut);margin:3px 0}.ev.w{color:var(--yel)}
.diag{color:var(--blu);margin:6px 0}
.cmd{color:var(--cmd);font-size:12px;white-space:pre-wrap;margin-left:14px}
.logbar{display:flex;align-items:center;gap:8px;margin:14px 0 6px}
.logbar h3{margin:0;font-size:12px;color:var(--mut);text-transform:uppercase;letter-spacing:.6px}
.mono.log{background:#0b0d12;border-radius:8px;padding:12px;font-size:12px;white-space:pre-wrap;
 word-break:break-all;max-height:46vh;overflow:auto}
.log .e{color:var(--red)}.log .nn{color:var(--mut)}
.empty{color:var(--grn);text-align:center;padding:40px}
</style></head><body>
<div class="app">
 <header>
  <span class="brand">Kube Oxente</span>
  <select id="ctxsel" onchange="switchCtx()"></select>
  <span class="badge" id="envb"></span>
  <select id="nsGlobal" onchange="setNS(this.value)" title="Filtrar namespace"></select>
  <span class="right">
   <span id="gen"></span>
   <span class="tgl" id="autob" onclick="toggleAuto()">&#x21bb; auto: on</span>
   <button onclick="exportReport()" title="Exportar snapshot HTML">&#10515; exportar</button>
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
 {id:"overview",  ico:"&#9632;", label:"Overview"},
 {id:"problems",  ico:"&#9888;", label:"Problemas"},
 {id:"pods",      ico:"&#9673;", label:"Pods"},
 {id:"workloads", ico:"&#9707;", label:"Workloads"},
 {id:"services",  ico:"&#8644;", label:"Services"},
 {id:"namespaces",ico:"&#9783;", label:"Namespaces"},
 {id:"events",    ico:"&#9993;", label:"Eventos"},
 {id:"nodes",     ico:"&#9636;", label:"N&oacute;s"},
 {id:"network",   ico:"&#8599;", label:"Rede"},
 {id:"arch",      ico:"&#9672;", label:"Arquitetura"},
 {id:"cronjobs",  ico:"&#9711;", label:"CronJobs"},
 {id:"hpas",      ico:"&#8645;", label:"HPA"},
 {id:"pvcs",      ico:"&#9644;", label:"PVCs"},
 {id:"configmaps",ico:"&#9783;", label:"ConfigMaps"},
 {id:"quotas",    ico:"&#9632;", label:"Quotas"}
];

var view="overview",auto=true,timer=null,counts={},podsCache=[],filt={env:"",q:"",st:"",app:""},renderSeq=0;
var globalNS="",metCache={pods:[],nodes:[],_pm:{},_nm:{}},rstHist={},liveLogSrc=null;
var _detNS="",_detName="";

function esc(s){return(s==null?"":""+s).replace(/[&<>"]/g,function(c){return{"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}[c]})}
function api(p){return fetch(p).then(function(r){return r.json()})}
function envBadge(e){return'<span class="badge '+(e||"other")+'">'+esc(e||"other")+'</span>'}
function sev(s){return'<span class="dot s'+s+'">&#9679;</span>'}
function applyNS(arr,f){arr=arr||[];f=f||"ns";if(!globalNS)return arr;return arr.filter(function(x){return x[f]===globalNS})}

/* ---- nav ---- */
function renderNav(){
 document.getElementById("nav").innerHTML=VIEWS.map(function(v){
  var badge=counts[v.id]!=null?'<span class="cnt'+((v.id==="problems"||v.id==="cronjobs")&&counts[v.id]>0?" red":"")+'">'+counts[v.id]+"</span>":"";
  return'<a class="'+(v.id===view?"on":"")+'" onclick="nav(\''+v.id+'\')"><span class="ico">'+v.ico+"</span>"+v.label+badge+"</a>";
 }).join("");
}
function nav(v){view=v;if(location.hash.slice(1).split(":")[0]!==v)history.replaceState(null,"","#"+v);renderNav();schedule();render()}

/* ---- contexts ---- */
function loadContexts(){
 api("/api/contexts").then(function(d){
  var s=document.getElementById("ctxsel");
  s.innerHTML=(d.all||[]).map(function(c){return'<option'+(c===d.active?" selected":"")+">"+esc(c)+"</option>"}).join("")||
   "<option>"+esc(d.active||"(sem contexto)")+"</option>";
 });
}
function switchCtx(){
 var c=document.getElementById("ctxsel").value;
 api("/api/setcontext?name="+encodeURIComponent(c)).then(function(){counts={};loadNsFilter();loadMetrics();refresh()});
}

/* ---- namespace filter ---- */
function loadNsFilter(){
 api("/api/namespaces").then(function(d){
  var s=document.getElementById("nsGlobal");if(!s)return;
  var cur=globalNS;
  s.innerHTML='<option value="">todos namespaces</option>'+
   d.map(function(n){return'<option'+(n.namespace===cur?" selected":"")+' value="'+esc(n.namespace)+'">'+esc(n.namespace)+"</option>"}).join("");
  s.value=cur;
 });
}
function setNS(v){globalNS=v;render()}

/* ---- metrics (kubectl top) ---- */
function loadMetrics(){
 api("/api/metrics").then(function(d){
  if(!d||d.err)return;
  metCache=d;metCache._pm={};metCache._nm={};
  (d.pods||[]).forEach(function(p){metCache._pm[p.ns+"/"+p.name]=p});
  (d.nodes||[]).forEach(function(n){metCache._nm[n.name]=n});
  if(_detName)updateDetMetrics(_detNS,_detName);
 });
}
function pMet(ns,name){return metCache._pm[ns+"/"+name]||null}
function nMet(name){return metCache._nm[name]||null}
function fmtM(val,pct){
 if(!val&&!pct)return'<span class="muted">&#8210;</span>';
 var n=parseInt(pct||"0"),cls=n>95?"crit":n>80?"hi":"";
 return'<span class="met'+(cls?" "+cls:"")+'">'+esc(val||pct)+(val&&pct?" <span style=\"opacity:.6\">("+esc(pct)+")</span>":"")+"</span>";
}

/* ---- sparklines (restart history) ---- */
function trackRst(pods){
 var t=Date.now();
 pods.forEach(function(p){
  var k=p.ns+"/"+p.name;
  if(!rstHist[k])rstHist[k]=[];
  var h=rstHist[k];
  if(!h.length||h[h.length-1].r!==p.restarts)h.push({t:t,r:p.restarts});
  if(h.length>24)h.shift();
 });
}
function spark(ns,name){
 var h=rstHist[ns+"/"+name];if(!h||h.length<2)return"";
 var max=Math.max.apply(null,h.map(function(x){return x.r}));if(!max)return"";
 var W=36,H=14;
 var pts=h.map(function(x,i){return((i/(h.length-1))*W).toFixed(1)+","+(H-x.r/max*H).toFixed(1)}).join(" ");
 return'<svg class="spark" width="'+W+'" height="'+H+'"><polyline points="'+pts+'" fill="none" stroke="var(--yel)" stroke-width="1.5" stroke-linejoin="round"/></svg>';
}

/* ---- schedule / refresh ---- */
function toggleAuto(){auto=!auto;document.getElementById("autob").innerHTML="&#x21bb; auto: "+(auto?"on":"off");schedule()}
function schedule(){clearTimeout(timer);if(auto)timer=setTimeout(refresh,7000)}
function refresh(){render();if(auto)loadMetrics();schedule()}

/* ---- render dispatcher ---- */
function render(){
 renderNav();++renderSeq;
 var m=document.getElementById("main");
 var fns={overview:rOverview,problems:rProblems,pods:rPods,workloads:rWorkloads,
  services:rServices,namespaces:rNamespaces,events:rEvents,nodes:rNodes,
  network:rNetwork,cronjobs:rCronJobs,hpas:rHPAs,pvcs:rPVCs,
  configmaps:rConfigMaps,quotas:rQuotas,arch:function(m){if(window.rArch)window.rArch(m)}};
 (fns[view]||rOverview)(m);
}
function setGen(){document.getElementById("gen").textContent="atualizado "+new Date().toLocaleTimeString()}

/* ---- export ---- */
function exportReport(){
 var m=document.getElementById("main");
 var ctx=document.getElementById("ctxsel").value||"cluster";
 var ts=new Date().toLocaleString("pt-BR");
 var h='<!doctype html><html><head><meta charset="utf-8"><title>Kube Oxente — '+esc(ctx)+'</title>'+
  '<style>body{font:13px system-ui;background:#fff;color:#1a1a1a;padding:24px;max-width:1280px;margin:0 auto}'+
  'h1{margin:0 0 4px}p.ts{color:#888;font-size:12px;margin:0 0 20px}'+
  'table{width:100%;border-collapse:collapse;margin-bottom:20px;font-size:12px}'+
  'th,td{padding:6px 10px;border:1px solid #e2e2e2;text-align:left}th{background:#f5f7fa;font-weight:600;color:#555}'+
  'tr:nth-child(even)td{background:#fafafa}.s2{color:#d32}.s1{color:#f73}.s0,.ok{color:#292}.bad{color:#d32}'+
  '.badge{padding:1px 7px;border-radius:12px;font-size:11px;font-weight:600}'+
  '.badge.prod{background:#fde;color:#d32}.badge.staging{background:#ffe;color:#960}'+
  '.badge.dev{background:#e8f0fe;color:#36c}.badge.other{background:#f0f0f0;color:#666}'+
  '.pill{padding:1px 7px;border-radius:4px;background:#eee;font-size:11px}'+
  '.sec{border:1px solid #e2e2e2;border-radius:8px;padding:14px;margin-bottom:16px}'+
  '.sec h3{margin:0 0 10px;font-size:11px;color:#888;text-transform:uppercase;letter-spacing:.5px}'+
  '</style></head><body>'+
  '<h1>Kube Oxente — '+esc(ctx)+'</h1>'+
  '<p class="ts">Exportado em '+esc(ts)+(globalNS?" &middot; namespace: "+esc(globalNS):"")+"</p>"+
  m.innerHTML+"</body></html>";
 var a=document.createElement("a");
 a.href="data:text/html;charset=utf-8,"+encodeURIComponent(h);
 a.download="koxente-"+ctx.replace(/[^a-z0-9]/gi,"-")+".html";
 document.body.appendChild(a);a.click();document.body.removeChild(a);
}

/* ===================== RENDER FUNCTIONS ===================== */

function rOverview(m){var seq=renderSeq;
 api("/api/overview"+(globalNS?"?ns="+encodeURIComponent(globalNS):"")).then(function(d){
  if(renderSeq!==seq)return;setGen();
  var eb=document.getElementById("envb");
  if(eb){eb.className="badge "+(d.env||"other");eb.title=d.ctx||"";eb.textContent=d.clusterName||d.env||"other"}
  var p=d.pods||{};
  var html='<h2>Overview <span class="muted" style="font-size:13px">'+esc(d.clusterName||d.ctx)
   +(globalNS?" &middot; "+esc(globalNS):"")+"</span></h2>";
  if(d.err){m.innerHTML=html+'<div class="spin">erro: '+esc(d.err)+"</div>";return}
  html+='<div class="cards">'+
   card(p.Total,"pods")+card(p.Running,"running","grn")+
   card(p.Pending,"pending",p.Pending?"yel":"")+
   card((p.Failed||0)+(p.Prob||0),"com falha",(p.Failed||0)+(p.Prob||0)?"red":"")+
   card(d.nsCount,"namespaces")+
   card(d.nodesTotal,"n&oacute;s"+(d.nodesDown?" ("+d.nodesDown+" down)":""),d.nodesDown?"red":"")+
   card(d.probApps,"apps c/ falha",d.probApps?"red":"")+"</div>";
  var tot=p.OK+p.Warn+p.Prob||1;
  html+='<div class="sec"><h3>Sa&uacute;de dos pods</h3><div class="bar">'+
   barseg(p.OK,tot,"var(--grn)")+barseg(p.Warn,tot,"var(--yel)")+barseg(p.Prob,tot,"var(--red)")+
   '</div><div class="legend">'+leg("var(--grn)","ok "+p.OK)+leg("var(--yel)","aten&ccedil;&atilde;o "+p.Warn)+leg("var(--red)","problema "+p.Prob)+"</div></div>";
  if(d.envSplit&&d.envSplit.length){
   html+='<div class="sec"><h3>Pods por ambiente</h3>';
   d.envSplit.forEach(function(e){html+='<div class="wl"><span>'+envBadge(e.label)+"</span><b>"+e.value+"</b></div>"});
   html+="</div>";
  }
  html+='<div class="sec"><h3>Workloads prontos</h3>';
  (d.workloads||[]).forEach(function(w){var ok=w.ready>=w.total;
   html+='<div class="wl"><span>'+esc(w.kind)+'s</span><span class="'+(ok?"ok":"bad")+'">'+w.ready+"/"+w.total+" prontos</span></div>"});
  html+="</div>";
  html+='<div class="sec"><h3>N&oacute;s</h3><div class="muted mono">'+(d.nodes||[]).map(esc).join(" &middot; ")+"</div></div>";
  m.innerHTML=html;
 });
}
function card(n,l,cls){return'<div class="card '+(cls||"")+'"><div class="n">'+n+'</div><div class="l">'+l+"</div></div>"}
function barseg(v,tot,c){return'<i style="width:'+(v/tot*100).toFixed(1)+'%;background:'+c+'"></i>'}
function leg(c,t){return'<span><span class="dotc" style="background:'+c+'"></span>'+t+"</span>"}

function rProblems(m){var seq=renderSeq;
 api("/api/problems").then(function(d){
  if(renderSeq!==seq)return;setGen();
  d=applyNS(d,"ns");counts.problems=d.length;renderNav();
  if(!d.length){m.innerHTML='<h2>Problemas</h2><div class="empty">&#10003; nenhuma falha'+(globalNS?" em "+esc(globalNS):"")+"</div>";return}
  m.innerHTML='<h2>Problemas <span class="muted" style="font-size:13px">'+d.length+" apps</span></h2>"+
   '<table><thead><tr><th></th><th>Aplica&ccedil;&atilde;o</th><th>Amb.</th><th>Motivo</th><th>Diagn&oacute;stico prov&aacute;vel</th></tr></thead><tbody>'+
   d.map(function(a){
    var grp=a.podCount>1?' <span class="pill">'+a.podCount+" pods</span>":"";
    var last=a.lastExit?" &middot; &uacute;lt. restart "+esc(a.lastExit)+" atr&aacute;s":"";
    return'<tr class="clk" onclick="openDetail(\''+esc(a.ns)+'\',\''+esc(a.samplePod)+'\',\''+esc(a.sampleReason)+'\')">'
     +'<td>'+sev(a.severity)+'</td><td><b>'+esc(a.app)+"</b>"+grp+'<div class="muted" style="font-size:12px">'+esc(a.ns)+'</div></td>'
     +'<td>'+envBadge(a.env)+'</td><td class="s'+a.severity+'">'+esc(a.reason)+'<div class="muted" style="font-size:11px">restarts '+a.restarts+last+"</div></td>"
     +'<td class="diagln">'+esc(a.primaryDiag)+"</td></tr>";
   }).join("")+"</tbody></table>";
 });
}

function rPods(m){var seq=renderSeq;
 api("/api/pods").then(function(d){
  if(renderSeq!==seq)return;setGen();podsCache=d;trackRst(d);
  var envOpt=function(v,l){return'<option value="'+v+'"'+(filt.env===v?" selected":"")+'>'+l+"</option>"};
  var stOpt=function(v,l){return'<option value="'+v+'"'+(filt.st===v?" selected":"")+'>'+l+"</option>"};
  var appOpt=function(v,l){return'<option value="'+esc(v)+'"'+(filt.app===v?" selected":"")+'>'+esc(l)+"</option>"};
  var scoped=globalNS?d.filter(function(p){return p.ns===globalNS}):d;
  var seenApp={},apps=[];
  scoped.forEach(function(p){if(p.app&&!seenApp[p.app]){seenApp[p.app]=true;apps.push(p.app)}});
  apps.sort();
  if(filt.app&&apps.indexOf(filt.app)<0)filt.app="";
  m.innerHTML='<h2>Pods</h2><div class="filters">'
   +'<select id="fapp" onchange="filt.app=this.value;drawPods()">'+appOpt("","todas aplica&ccedil;&otilde;es")+apps.map(function(a){return appOpt(a,a)}).join("")+'</select>'
   +'<select id="fenv" onchange="filt.env=this.value;drawPods()">'+envOpt("","todos ambientes")+envOpt("prod","prod")+envOpt("staging","staging")+envOpt("dev","dev")+envOpt("other","other")+'</select>'
   +'<select id="fst" onchange="filt.st=this.value;drawPods()">'+stOpt("","todos status")+stOpt("2","problema")+stOpt("1","aten&ccedil;&atilde;o")+stOpt("0","ok")+'</select>'
   +'<input type="text" id="fq" placeholder="filtrar por nome/namespace&#8230;" value="'+esc(filt.q)+'" oninput="filt.q=this.value;drawPods()">'
   +'<span class="muted" id="pcount"></span></div>'
   +'<table><thead><tr><th></th><th>Namespace</th><th>Pod</th><th>Amb.</th><th>Status</th>'
   +'<th>Ready</th><th>Restarts</th><th>CPU</th><th>Mem</th><th>Idade</th><th>N&oacute;</th>'
   +'</tr></thead><tbody id="pbody"></tbody></table>';
  drawPods();
 });
}
function drawPods(){
 var d=podsCache.filter(function(p){
  if(globalNS&&p.ns!==globalNS)return false;
  if(filt.app&&p.app!==filt.app)return false;
  if(filt.env&&p.env!==filt.env)return false;
  if(filt.st!==""&&(""+p.severity)!==filt.st)return false;
  if(filt.q){var q=filt.q.toLowerCase();if((p.name+" "+p.ns).toLowerCase().indexOf(q)<0)return false}
  return true;
 });
 document.getElementById("pcount").textContent=d.length+" de "+podsCache.length;
 document.getElementById("pbody").innerHTML=d.map(function(p){
  var met=pMet(p.ns,p.name);
  return'<tr class="clk" onclick="openDetail(\''+esc(p.ns)+'\',\''+esc(p.name)+'\',\''+esc(p.reason)+'\')">'
   +'<td>'+sev(p.severity)+'</td><td class="muted">'+esc(p.ns)+'</td><td>'+esc(p.name)+'</td>'
   +'<td>'+envBadge(p.env)+'</td><td class="s'+p.severity+'">'+esc(p.reason)+'</td>'
   +'<td>'+p.ready+"/"+p.total+'</td>'
   +'<td>'+p.restarts+spark(p.ns,p.name)+'</td>'
   +'<td>'+(met?fmtM(met.cpu):'<span class="muted">&#8210;</span>')+'</td>'
   +'<td>'+(met?fmtM(met.memory):'<span class="muted">&#8210;</span>')+'</td>'
   +'<td>'+esc(p.age)+'</td><td class="muted mono" style="font-size:11px">'+esc(p.node)+'</td></tr>';
 }).join("")||'<tr><td colspan="11" class="empty">nenhum pod</td></tr>';
}

function rWorkloads(m){var seq=renderSeq;
 api("/api/workloads").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");
  m.innerHTML='<h2>Workloads</h2>'
   +'<table><thead><tr><th>Tipo</th><th>Namespace</th><th>Nome</th><th>Amb.</th><th>Ready</th></tr></thead><tbody>'
   +d.map(function(w){return'<tr><td><span class="pill">'+esc(w.kind)+'</span></td><td class="muted">'+esc(w.ns)+'</td><td>'+esc(w.name)+'</td>'
    +'<td>'+envBadge(w.env)+'</td><td class="'+(w.ok?"ok":"bad")+'">'+w.ready+"/"+w.desired+'</td></tr>'}).join("")
   +(d.length?"":'<tr><td colspan="5" class="empty">nenhum workload</td></tr>')
   +'</tbody></table>';
 });
}

function rServices(m){var seq=renderSeq;
 api("/api/services").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");
  m.innerHTML='<h2>Services</h2>'
   +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Tipo</th><th>ClusterIP</th><th>Portas</th><th>Endpoints</th><th>Amb.</th></tr></thead><tbody>'
   +d.map(function(s){
    var epTxt=(s.readyEps||s.notReadyEps)
     ?('<span class="ok">'+s.readyEps+' pronto</span>'+(s.notReadyEps?' <span class="s1">+'+s.notReadyEps+' n&atilde;o pronto</span>':""))
     :(s.orphan?'<span class="s2">sem endpoint</span>':'<span class="muted">&#8210;</span>');
    return'<tr><td class="muted">'+esc(s.ns)+'</td><td>'+esc(s.name)+'</td><td>'+esc(s.type)+'</td>'
    +'<td class="mono muted">'+esc(s.ip)+'</td><td class="mono">'+esc(s.ports)+'</td><td>'+epTxt+'</td><td>'+envBadge(s.env)+'</td></tr>'}).join("")
   +(d.length?"":'<tr><td colspan="7" class="empty">nenhum service</td></tr>')
   +'</tbody></table>';
 });
}

function rNamespaces(m){var seq=renderSeq;
 api("/api/namespaces").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"namespace");counts.namespaces=d.length;renderNav();
  m.innerHTML='<h2>Namespaces</h2>'
   +'<table><thead><tr><th>Namespace</th><th>Amb.</th><th>Pods</th><th>OK</th><th>Aten&ccedil;&atilde;o</th><th>Problema</th></tr></thead><tbody>'
   +d.map(function(n){return'<tr><td>'+esc(n.namespace)+'</td><td>'+envBadge(n.env)+'</td><td>'+n.pods+'</td>'
    +'<td class="ok">'+n.running+'</td><td class="s1">'+(n.warn||"")+'</td><td class="s2">'+(n.prob||"")+'</td></tr>'}).join("")
   +(d.length?"":'<tr><td colspan="6" class="empty">nenhum namespace</td></tr>')
   +'</tbody></table>';
 });
}

function rEvents(m){var seq=renderSeq;
 api("/api/events").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");counts.events=d.length;renderNav();
  if(!d.length){m.innerHTML='<h2>Eventos</h2><div class="empty">&#10003; sem Warning'+(globalNS?" em "+esc(globalNS):"")+"</div>";return}
  m.innerHTML='<h2>Eventos <span class="muted" style="font-size:13px">Warning</span></h2>'
   +'<table><thead><tr><th>Namespace</th><th>Motivo</th><th>Objeto</th><th>Mensagem</th><th>Qtd</th></tr></thead><tbody>'
   +d.map(function(e){return'<tr><td class="muted">'+esc(e.ns)+'</td><td class="s1">'+esc(e.reason)+'</td>'
    +'<td class="mono" style="font-size:12px">'+esc(e.object)+'</td>'
    +'<td class="muted" style="font-size:12px">'+esc(e.message)+'</td><td>'+e.count+'</td></tr>'}).join("")
   +'</tbody></table>';
 });
}

function rNodes(m){var seq=renderSeq;
 api("/api/nodes").then(function(d){
  if(renderSeq!==seq)return;setGen();counts.nodes=d.length;renderNav();
  m.innerHTML='<h2>N&oacute;s</h2>'
   +'<table><thead><tr><th></th><th>Nome</th><th>Tipo</th><th>Capacidade</th><th>Zona</th><th>CPU</th><th>Mem</th><th>Idade</th></tr></thead><tbody>'
   +d.map(function(n){
    var met=nMet(n.name);
    return'<tr><td><span class="dot '+(n.ready?"s0":"s2")+'">&#9679;</span></td>'
     +'<td class="mono" style="font-size:12px">'+esc(n.name)+'</td>'
     +'<td>'+esc(n.type||"-")+'</td><td>'+esc(n.cap||"-")+'</td><td class="muted">'+esc(n.zone||"-")+'</td>'
     +'<td>'+(met?fmtM(met.cpu,met.cpuPct):'<span class="muted">&#8210;</span>')+'</td>'
     +'<td>'+(met?fmtM(met.memory,met.memPct):'<span class="muted">&#8210;</span>')+'</td>'
     +'<td>'+esc(n.age)+'</td></tr>';
   }).join("")
   +(d.length?"":'<tr><td colspan="8" class="empty">nenhum n&oacute;</td></tr>')
   +'</tbody></table>';
 });
}

function rNetwork(m){var seq=renderSeq;
 Promise.all([api("/api/ingresses"),api("/api/services"),api("/api/netinfo")]).then(function(res){
  if(renderSeq!==seq)return;setGen();
  var ing=applyNS(res[0],"ns"),svc=applyNS(res[1],"ns"),net=res[2];
  counts.network=ing.length;renderNav();

  var probSvc=svc.filter(function(s){return s.orphan||(s.portMismatch&&s.portMismatch.length)});

  var h='<h2>Rede</h2>';

  h+='<h3 class="muted">Endpoints <span class="muted" style="font-size:12px">'+probSvc.length+" com problema</span></h3>";
  if(!probSvc.length){
   h+='<div class="empty">&#10003; nenhum Service &oacute;rf&atilde;o ou com mismatch de porta'+(globalNS?" em "+esc(globalNS):"")+"</div>";
  }else{
   h+='<table><thead><tr><th>Namespace</th><th>Service</th><th>Problema</th><th>Amb.</th></tr></thead><tbody>'
    +probSvc.map(function(s){
     var probs=[];
     if(s.orphan)probs.push('<span class="s2">sem endpoint pronto (selector n&atilde;o casa com nenhum pod saud&aacute;vel)</span>');
     if(s.portMismatch&&s.portMismatch.length)probs.push('<span class="s1">porta(s) '+s.portMismatch.map(esc).join(", ")+' sem container correspondente</span>');
     return'<tr><td class="muted">'+esc(s.ns)+'</td><td>'+esc(s.name)+'</td><td>'+probs.join("<br>")+'</td><td>'+envBadge(s.env)+'</td></tr>';
    }).join("")
    +'</tbody></table>';
  }

  h+='<h3 class="muted" style="margin-top:18px">NetworkPolicy</h3>';
  if(!net.netpolicyTotal){
   h+='<div class="empty">nenhuma NetworkPolicy encontrada no cluster &mdash; sem um agente de enforcement (Calico/Cilium/VPC CNI NetworkPolicy), todo o tr&aacute;fego entre pods &eacute; permitido por padr&atilde;o</div>';
  }else{
   var npRows=applyNS(net.netpolicy,"ns");
   var unprot=npRows.filter(function(n){return n.unprotected&&n.pods>0}).length;
   h+='<div class="muted" style="font-size:12px;margin-bottom:6px">'+net.netpolicyTotal+' policy(ies) no cluster'
    +(unprot?' &middot; <span class="s1">'+unprot+' namespace(s) sem cobertura</span>':"")+'</div>'
    +'<table><thead><tr><th>Namespace</th><th>Policies</th><th>Pods</th><th>Cobertura</th><th>Amb.</th></tr></thead><tbody>'
    +npRows.map(function(n){
     var cov=n.unprotected?'<span class="s1">sem policy (tr&aacute;fego aberto)</span>':'<span class="ok">'+n.policies+" policy(ies)</span>";
     return'<tr><td class="muted">'+esc(n.ns)+'</td><td>'+n.policies+'</td><td>'+n.pods+'</td><td>'+cov+'</td><td>'+envBadge(n.env)+'</td></tr>';
    }).join("")
    +'</tbody></table>';
  }

  h+='<h3 class="muted" style="margin-top:18px">CNI (VPC CNI / aws-node)</h3>';
  if(!net.cni||!net.cni.found){
   h+='<div class="empty">DaemonSet aws-node n&atilde;o encontrado (n&atilde;o &eacute; EKS com VPC CNI, ou sem permiss&atilde;o)</div>';
  }else{
   var cniOk=net.cni.readyNodes>=net.cni.desiredNodes;
   h+='<div class="wl"><span>Pods aws-node prontos</span><span class="'+(cniOk?"ok":"bad")+'">'+net.cni.readyNodes+"/"+net.cni.desiredNodes+'</span></div>'
    +'<div class="wl"><span>External SNAT</span><span class="mono">'+esc(net.cni.externalSnat||"false (padr&atilde;o)")+'</span></div>'
    +'<div class="wl"><span>WARM_IP_TARGET</span><span class="mono">'+esc(net.cni.warmIpTarget||"&#8210;")+'</span></div>'
    +'<div class="wl"><span>MINIMUM_IP_TARGET</span><span class="mono">'+esc(net.cni.minimumIpTarget||"&#8210;")+'</span></div>'
    +'<div class="wl"><span>WARM_ENI_TARGET</span><span class="mono">'+esc(net.cni.warmEniTarget||"&#8210;")+'</span></div>';
  }

  h+='<h3 class="muted" style="margin-top:18px">CoreDNS</h3>';
  if(!net.dns||!net.dns.found){
   h+='<div class="empty">Deployment coredns n&atilde;o encontrado</div>';
  }else{
   var dnsOk=net.dns.ready>=net.dns.desired;
   h+='<div class="wl"><span>R&eacute;plicas prontas</span><span class="'+(dnsOk?"ok":"bad")+'">'+net.dns.ready+"/"+net.dns.desired+'</span></div>'
    +'<div class="wl"><span>Forwarders</span><span class="mono">'+esc(net.dns.forwarders||"padr&atilde;o (sem forward customizado)")+'</span></div>';
  }

  var tgb=applyNS(net.targetGroupBindings||[],"ns");
  if(tgb.length){
   h+='<h3 class="muted" style="margin-top:18px">Target Group Bindings <span class="muted" style="font-size:12px">'+tgb.length+"</span></h3>"
    +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Service</th><th>Tipo</th><th>Target Group ARN</th></tr></thead><tbody>'
    +tgb.map(function(t){return'<tr><td class="muted">'+esc(t.ns)+'</td><td>'+esc(t.name)+'</td><td>'+esc(t.serviceName)+'</td>'
     +'<td class="mono">'+esc(t.targetType)+'</td><td class="mono" style="font-size:11px">'+esc(t.targetGroupArn)+'</td></tr>'}).join("")
    +'</tbody></table>';
  }

  var sgp=applyNS(net.securityGroupPolicies||[],"ns");
  if(sgp.length){
   h+='<h3 class="muted" style="margin-top:18px">Security Group Policies <span class="muted" style="font-size:12px">'+sgp.length+"</span></h3>"
    +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Security Groups</th></tr></thead><tbody>'
    +sgp.map(function(s){return'<tr><td class="muted">'+esc(s.ns)+'</td><td>'+esc(s.name)+'</td>'
     +'<td class="mono">'+esc((s.securityGroups||[]).join(", "))+'</td></tr>'}).join("")
    +'</tbody></table>';
  }

  h+='<h3 class="muted" style="margin-top:18px">Ingress <span class="muted" style="font-size:12px">'+ing.length+" entradas</span></h3>"
   +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Classe</th><th>Hosts</th><th>Endere&ccedil;o</th><th>Portas</th><th>Amb.</th><th>Idade</th></tr></thead><tbody>'
   +ing.map(function(r){return'<tr><td class="muted">'+esc(r.ns)+'</td><td>'+esc(r.name)+'</td>'
    +'<td class="muted mono">'+esc(r.class||"-")+'</td>'
    +'<td class="mono" style="font-size:12px">'+esc(r.hosts||"-")+'</td>'
    +'<td class="muted mono" style="font-size:12px">'+esc(r.address||"pendente")+'</td>'
    +'<td class="mono">'+esc(r.ports)+'</td><td>'+envBadge(r.env)+'</td><td>'+esc(r.age)+'</td></tr>'}).join("")
   +(ing.length?"":'<tr><td colspan="8" class="empty">nenhum ingress</td></tr>')
   +'</tbody></table>';

  m.innerHTML=h;
 });
}

function rCronJobs(m){var seq=renderSeq;
 api("/api/cronjobs").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");
  var probs=d.filter(function(r){return r.severity>0}).length;
  counts.cronjobs=probs;renderNav();
  m.innerHTML='<h2>CronJobs <span class="muted" style="font-size:13px">'+d.length+" total"
   +(probs?' &middot; <span class="bad">'+probs+" com problema</span>":"")+"</span></h2>"
   +'<table><thead><tr><th></th><th>Namespace</th><th>Nome</th><th>Schedule</th>'
   +'<th>Ativo</th><th>&Uacute;lt. execu&ccedil;&atilde;o</th><th>&Uacute;lt. sucesso</th><th>Status</th><th>Amb.</th>'
   +'</tr></thead><tbody>'
   +d.map(function(r){
    var actStr=r.active>0?'<span class="ok">'+r.active+" rodando</span>":'<span class="muted">&#8210;</span>';
    var schedStr=r.lastSchedule?esc(r.lastSchedule)+" atr&aacute;s":'<span class="muted">nunca</span>';
    var succStr=r.lastSuccess?esc(r.lastSuccess)+" atr&aacute;s":'<span class="muted">nunca</span>';
    var probStr=r.problem?'<span class="s'+r.severity+'">'+esc(r.problem)+'</span>':'<span class="ok">ok</span>';
    return'<tr><td>'+sev(r.severity)+'</td>'
     +'<td class="muted">'+esc(r.ns)+'</td>'
     +'<td><b>'+esc(r.name)+'</b>'+(r.suspended?' <span class="pill">suspenso</span>':"")+'</td>'
     +'<td class="mono" style="font-size:12px">'+esc(r.schedule)+'</td>'
     +'<td>'+actStr+'</td><td>'+schedStr+'</td><td>'+succStr+'</td>'
     +'<td>'+probStr+'</td><td>'+envBadge(r.env)+'</td></tr>';
   }).join("")
   +(d.length?"":'<tr><td colspan="9" class="empty">nenhum cronjob</td></tr>')
   +'</tbody></table>';
 });
}

function rHPAs(m){var seq=renderSeq;
 api("/api/hpas").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");counts.hpas=d.length;renderNav();
  m.innerHTML='<h2>HPA <span class="muted" style="font-size:13px">'+d.length+" autoscalers</span></h2>"
   +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Refer&ecirc;ncia</th>'
   +'<th>Min</th><th>Max</th><th>Atual</th><th>Desejado</th><th>Amb.</th><th>Idade</th>'
   +'</tr></thead><tbody>'
   +d.map(function(r){return'<tr><td class="muted">'+esc(r.ns)+'</td><td>'+esc(r.name)+'</td>'
    +'<td class="muted mono" style="font-size:12px">'+esc(r.reference)+'</td>'
    +'<td>'+r.minReplicas+'</td><td>'+r.maxReplicas+'</td>'
    +'<td class="'+(r.ok?"ok":"bad")+'"><b>'+r.current+'</b></td>'
    +'<td>'+r.desired+'</td><td>'+envBadge(r.env)+'</td><td>'+esc(r.age)+'</td></tr>'}).join("")
   +(d.length?"":'<tr><td colspan="9" class="empty">nenhum HPA</td></tr>')
   +'</tbody></table>';
 });
}

function rPVCs(m){var seq=renderSeq;
 api("/api/pvcs").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");counts.pvcs=d.length;renderNav();
  m.innerHTML='<h2>PVCs <span class="muted" style="font-size:13px">'+d.length+" volumes</span></h2>"
   +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Status</th>'
   +'<th>Capacidade</th><th>Modos</th><th>StorageClass</th><th>Amb.</th><th>Idade</th>'
   +'</tr></thead><tbody>'
   +d.map(function(r){
    var stCls=r.status==="Bound"?"ok":r.status==="Pending"?"s1":"s2";
    return'<tr><td class="muted">'+esc(r.ns)+'</td><td>'+esc(r.name)+'</td>'
     +'<td class="'+stCls+'">'+esc(r.status)+'</td>'
     +'<td class="mono">'+esc(r.capacity||"&#8210;")+'</td>'
     +'<td class="mono muted">'+esc(r.accessModes||"&#8210;")+'</td>'
     +'<td class="muted mono" style="font-size:12px">'+esc(r.storageClass||"&#8210;")+'</td>'
     +'<td>'+envBadge(r.env)+'</td><td>'+esc(r.age)+'</td></tr>';
   }).join("")
   +(d.length?"":'<tr><td colspan="8" class="empty">nenhum PVC</td></tr>')
   +'</tbody></table>';
 });
}

function rConfigMaps(m){var seq=renderSeq;
 api("/api/configmaps").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");counts.configmaps=d.length;renderNav();
  m.innerHTML='<h2>ConfigMaps <span class="muted" style="font-size:13px">'+d.length+" configs</span></h2>"
   +'<table><thead><tr><th>Namespace</th><th>Nome</th><th>Chaves</th><th>Amb.</th><th>Idade</th></tr></thead><tbody>'
   +d.map(function(r){return'<tr><td class="muted">'+esc(r.ns)+'</td><td>'+esc(r.name)+'</td>'
    +'<td><span class="pill">'+r.keys+" keys</span></td>"
    +'<td>'+envBadge(r.env)+'</td><td>'+esc(r.age)+'</td></tr>'}).join("")
   +(d.length?"":'<tr><td colspan="5" class="empty">nenhum configmap</td></tr>')
   +'</tbody></table>';
 });
}

function rQuotas(m){var seq=renderSeq;
 api("/api/quotas").then(function(d){
  if(renderSeq!==seq)return;setGen();d=applyNS(d,"ns");counts.quotas=d.length;renderNav();
  if(!d.length){m.innerHTML='<h2>Quotas</h2><div class="empty">nenhuma resource quota configurada'+(globalNS?" em "+esc(globalNS):"")+"</div>";return}
  var html='<h2>Quotas <span class="muted" style="font-size:13px">'+d.length+" quotas</span></h2>";
  d.forEach(function(q){
   html+='<div class="sec"><h3>'+esc(q.ns)+" / "+esc(q.name)+"&nbsp;&nbsp;"+envBadge(q.env)+"</h3>"
    +'<table><thead><tr><th>Recurso</th><th>Usado</th><th>Limite</th><th>Uso</th></tr></thead><tbody>';
   Object.keys(q.resources||{}).sort().forEach(function(k){
    var r=q.resources[k];
    var used=r.used||"0",hard=r.hard||"-";
    var bar="";
    var uN=parseFloat(used),hN=parseFloat(hard);
    if(!isNaN(uN)&&!isNaN(hN)&&hN>0){
     var pct=Math.min(Math.round(uN/hN*100),100);
     var c=pct>90?"var(--red)":pct>70?"var(--yel)":"var(--grn)";
     bar='<span class="qbar"><i style="width:'+pct+'%;background:'+c+'"></i></span> '+pct+"%";
    }
    html+='<tr><td class="mono" style="font-size:12px">'+esc(k)+'</td>'
     +'<td>'+esc(used)+'</td><td>'+esc(hard)+'</td><td>'+bar+'</td></tr>';
   });
   html+="</tbody></table></div>";
  });
  m.innerHTML=html;
 });
}

/* ===================== DETAIL MODAL ===================== */

function openDetail(ns,name,reason){
 stopLiveLog();
 _detNS=ns;_detName=name;
 document.getElementById("ov").classList.add("show");
 var md=document.getElementById("modal");md.classList.add("show");
 document.getElementById("mh").innerHTML='<div><b>'+esc(name)+'</b>'
  +'<div class="muted" style="font-size:12px">'+esc(ns)+" &middot; "+esc(reason)+'</div>'
  +'<div style="font-size:12px;margin-top:2px" id="mMet"></div></div>'
  +'<span class="x" onclick="closeDetail()">&times;</span>';
 updateDetMetrics(ns,name);
 document.getElementById("mb").innerHTML='<div class="spin">carregando&#8230;</div>';
 api("/api/detail?ns="+encodeURIComponent(ns)+"&name="+encodeURIComponent(name)+"&reason="+encodeURIComponent(reason))
  .then(function(d){
   var h="";
   if(d.diags&&d.diags.length){
    h+='<h3 class="muted">Diagn&oacute;stico prov&aacute;vel</h3>';
    d.diags.forEach(function(g){
     h+='<div class="diag">&rarr; '+esc(g.title)+"</div>";
     (g.cmds||[]).forEach(function(c){h+='<div class="cmd mono">'+esc(c)+"</div>"});
    });
   }
   if(d.events&&d.events.length){
    h+='<h3 class="muted" style="margin-top:14px">Eventos</h3>';
    d.events.forEach(function(e){
     h+='<div class="ev'+(e.type==="Warning"?" w":"")+'">&#9656; '+esc(e.reason)+": "+esc(e.message)
      +' <span style="opacity:.6">('+esc(e.age)+")</span></div>";
    });
   }
   if(d.secrets&&d.secrets.length){
    h+='<h3 class="muted" style="margin-top:14px">Secrets</h3>';
    d.secrets.forEach(function(s){
     h+='<div class="diag">&rarr; '+esc(s.secret)+"</div>";
     if(s.denied){
      h+='<div class="cmd mono muted">(sem permiss&atilde;o de leitura neste Secret)</div>';
     }else if(!s.keys||!s.keys.length){
      h+='<div class="cmd mono muted">(nenhuma chave encontrada)</div>';
     }else{
      s.keys.forEach(function(k){h+='<div class="cmd mono">'+esc(k)+"=xxx</div>"});
     }
    });
   }
   h+='<div class="logbar"><h3 class="muted">Logs</h3>'
    +'<button id="btnLive" class="btn-live" onclick="startLiveLog()" style="font-size:11px;padding:3px 9px">&#9654; ao vivo</button>'
    +'<button id="btnStop" class="btn-stop" onclick="stopLiveLog()" style="font-size:11px;padding:3px 9px;display:none">&#9632; parar</button>'
    +"</div>"
    +'<div class="mono log" id="logbox">'
    +((d.logs&&d.logs.length)
      ?d.logs.map(function(l){return'<span class="'+(l.isErr?"e":"nn")+'">'+esc(l.text)+"</span>"}).join("\n")
      :'<span class="nn">(sem logs)</span>')
    +"</div>";
   document.getElementById("mb").innerHTML=h;
  });
}

function startLiveLog(){
 stopLiveLog();
 var box=document.getElementById("logbox");if(!box)return;
 box.innerHTML='<span class="nn">Conectando&#8230;</span>\n';
 document.getElementById("btnLive").style.display="none";
 document.getElementById("btnStop").style.display="";
 liveLogSrc=new EventSource("/api/logstream?ns="+encodeURIComponent(_detNS)+"&name="+encodeURIComponent(_detName));
 liveLogSrc.onmessage=function(e){
  var line=JSON.parse(e.data);
  var isErr=/error|fatal|panic|exception|denied|refused|timeout/i.test(line);
  var s=document.createElement("span");s.className=isErr?"e":"nn";s.textContent=line;
  box.appendChild(s);box.appendChild(document.createTextNode("\n"));
  var atBottom=box.scrollTop>=box.scrollHeight-box.clientHeight-60;
  if(atBottom)box.scrollTop=box.scrollHeight;
 };
 liveLogSrc.onerror=function(){stopLiveLog()};
}
function stopLiveLog(){
 if(liveLogSrc){liveLogSrc.close();liveLogSrc=null}
 var bl=document.getElementById("btnLive"),bs=document.getElementById("btnStop");
 if(bl)bl.style.display="";if(bs)bs.style.display="none";
}
function closeDetail(){
 stopLiveLog();
 _detNS="";_detName="";
 document.getElementById("ov").classList.remove("show");
 document.getElementById("modal").classList.remove("show");
}
function updateDetMetrics(ns,name){
 var el=document.getElementById("mMet");if(!el)return;
 var met=pMet(ns,name);
 el.innerHTML=met?("CPU "+fmtM(met.cpu)+' <span class="muted">&middot;</span> Mem '+fmtM(met.memory))
  :'<span class="muted">sem m&eacute;tricas (kubectl top)</span>';
}
document.addEventListener("keydown",function(e){if(e.key==="Escape")closeDetail()});

/* ===================== INIT ===================== */
function hashView(){var hv=location.hash.slice(1).split(":")[0];return VIEWS.some(function(v){return v.id===hv})?hv:""}
(function(){var hv=hashView();if(hv)view=hv})();
window.addEventListener("hashchange",function(){var hv=hashView();if(hv&&hv!==view)nav(hv)});
loadContexts();loadNsFilter();loadMetrics();render();schedule();
</script></body></html>`
