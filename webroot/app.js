
/* ===== 工具 ===== */
let LANG=localStorage.getItem('jc_lang')||'zh';
const I18N={zh:{nav_home:'首页',nav_clean:'清理',nav_settings:'设置',sub:'NEON HUD v3',
  free:'可用空间',scan_now:'⚡ 开始扫描',
  scanning:'深度扫描',review:'清理清单',cleaning:'正在清理…',done:'清理完成',
  settings:'设置',cats:'清理分类',appearance:'外观',tools:'工具',rules:'规则',system:'系统',
  theme:'主题',lang:'语言',trash:'回收站',
  save:'保存',send:'发送',add:'添加',confirm:'确认清理',back:'返回首页',rescan:'再次扫描',
  execute:'执行',refresh:'刷新',
  cat_cache:'应用缓存',cat_junk:'系统垃圾',cat_apk:'安装包',cat_zip:'压缩包',cat_thumb:'缩略图',
  cat_log:'日志',cat_temp:'临时文件',cat_uninst:'卸载残留',cat_zero:'空文件',cat_empty:'空文件夹',
  cat_social:'社交专项',cat_sqlite:'SQLite',cat_big:'大文件',files:'文件',count:'个',last_scan:'上次扫描',no_scan:'暂无扫描·点击开始体检',days_ago:'天前',next_clean:'下次清理',hours_after:'小时后',every:'每',hours:'小时',daily:'每日',cleaned_done:'清理完成，释放',selected:'已选',items:'项',guard_on:'守护中'},
  en:{nav_home:'Home',nav_clean:'Clean',nav_settings:'Settings',sub:'NEON HUD v3',
  free:'Available',scan_now:'⚡ Start Scan',
  scanning:'Deep Scan',review:'Review',cleaning:'Cleaning…',done:'Done',
  settings:'Settings',cats:'Categories',appearance:'Appearance',tools:'Tools',rules:'Rules',system:'System',
  theme:'Theme',lang:'Language',trash:'Trash',
  save:'Save',send:'Send',add:'Add',confirm:'Confirm Clean',back:'Home',rescan:'Scan Again',
  execute:'Run',refresh:'Refresh',
  cat_cache:'App Cache',cat_junk:'System Junk',cat_apk:'APK',cat_zip:'Archive',cat_thumb:'Thumbnails',
  cat_log:'Logs',cat_temp:'Temps',cat_uninst:'Uninstall',cat_zero:'Empty Files',cat_empty:'Empty Folders',
  cat_social:'Social',cat_sqlite:'SQLite',cat_big:'Big Files',files:'files',count:'',last_scan:'Last scan',no_scan:'No scan data·Start now',days_ago:'d ago',next_clean:'Next clean',hours_after:'h later',every:'Every',hours:'h',daily:'Daily',cleaned_done:'Cleaned, freed',selected:'Selected',items:'items',guard_on:'Running'}};
function T(k){return (I18N[LANG]||I18N.zh)[k]||k;}
const $=id=>document.getElementById(id);
const CATS={cache:[T('cat_cache'),'🗄'],junk:[T('cat_junk'),'🧩'],apk:[T('cat_apk'),'📦'],zip:[T('cat_zip'),'🗜'],thumb:[T('cat_thumb'),'🖼'],log:[T('cat_log'),'📋'],temp:[T('cat_temp'),'⏳'],uninst:[T('cat_uninst'),'🗑'],zero:[T('cat_zero'),'📄'],empty:[T('cat_empty'),'📂'],social:[T('cat_social'),'💬'],sqlite:[T('cat_sqlite'),'🗃'],big:[T('cat_big'),'📈']};
const CAT_ORDER=['cache','junk','apk','zip','thumb','log','temp','uninst','empty','zero','social','sqlite'];
function fmtKB(kb){ if(kb>=1048576) return (kb/1048576).toFixed(1)+'GB'; if(kb>=1024) return (kb/1024).toFixed(1)+'MB'; return kb+'KB'; }
function escapeHtml(t){return String(t).replace(/[&<>"]/g,function(m){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[m]||m})}
let toastT; function toast(msg){
  let el=$('toast'); if(!el){ el=document.createElement('div'); el.className='toast'; el.id='toast'; document.body.appendChild(el); }
  el.textContent=msg; el.classList.add('show');
  clearTimeout(toastT); toastT=setTimeout(()=>el.classList.remove('show'),2200);
}
function showDialog(title,html){
  const d=document.createElement('div');
  d.style.cssText='position:fixed;inset:0;background:rgba(3,6,12,.65);backdrop-filter:blur(10px);-webkit-backdrop-filter:blur(10px);z-index:90;display:flex;align-items:center;justify-content:center;padding:20px;animation:fd .2s';
  d.onclick=function(e){ if(e.target===d) d.remove(); };
  d.innerHTML='<div style="width:100%;max-width:440px;max-height:80vh;overflow:auto;border-radius:20px;padding:18px;'+
    'background:linear-gradient(180deg,rgba(16,24,36,.92),rgba(8,14,22,.88));'+
    'border:1px solid rgba(0,229,200,.25);'+
    'box-shadow:inset 0 1px 0 rgba(255,255,255,.08), 0 20px 60px rgba(0,0,0,.6), 0 0 44px rgba(0,229,200,.08);'+
    'animation:pd .3s cubic-bezier(.32,.72,0,1)">'+
    '<div style="display:flex;justify-content:space-between;align-items:center;font-weight:700;margin-bottom:12px;font-size:15px">'+title+
    '<button style="background:none;border:none;color:var(--sub);font-size:18px;cursor:pointer;padding:2px 6px;border-radius:50%">✕</button></div>'+
    html+'<div style="margin-top:14px;text-align:center"><button class="sw on" style="width:100%;padding:9px">关闭</button></div></div>';
  d.querySelector('button').onclick=function(){ d.remove(); };
  document.body.appendChild(d);
}
function animNum(el,to,fmt){
  if(!el) return; const D=600,t0=performance.now();
  const step=t=>{ const p=Math.min(1,(t-t0)/D),e=1-Math.pow(1-p,3);
    el.textContent=(fmt||fmtKB)(Math.round(to*e)); if(p<1) requestAnimationFrame(step); };
  requestAnimationFrame(step);
}
document.addEventListener('pointerdown',function(e){
  const b=e.target.closest('.btn,.cta,.sw,.qitem,.catcard,.tab');
  if(!b) return; const r=b.getBoundingClientRect();
  const ink=document.createElement('span'); ink.className='ripple-ink';
  ink.style.cssText='left:'+(e.clientX-r.left)+'px;top:'+(e.clientY-r.top)+'px;width:20px;height:20px';
  b.appendChild(ink); setTimeout(()=>ink.remove(),560);
});
/* ===== API（KSU 桥 + fetch 双通道） ===== */
const API=location.protocol==='http:'?'http://127.0.0.1:46780':'';
async function api(p,o){
  const url=p.startsWith('/api')?API+p:p;
  if(window.ksu&&typeof window.ksu.exec==='function'){
    try{
      const CURL='/data/adb/modules/junkclean/bin/curl';
      const body=(o&&o.body)?o.body:'';
      const cmd=CURL+' -s -m 20 '+JSON.stringify(url)+(o&&o.method==='POST'?' -X POST -H "Content-Type: application/json" --data '+JSON.stringify(body):'');
      const r=await window.ksu.exec(cmd);
      const txt=typeof r==='string'?r:(r&&(r.stdout||r.result)||'');
      return JSON.parse(txt);
    }catch(e){ return {}; }
  }
  try{ const r=await fetch(url,o); return r.json(); }catch(e){ return {}; }
}
/* ===== Tab / View ===== */
function showTab(t){
  document.querySelectorAll('.tabview').forEach(v=>v.classList.remove('show'));
  $('tab-'+t).classList.add('show');
  document.querySelectorAll('.tab').forEach(b=>b.classList.toggle('on',b.dataset.tab===t));
}
function gotoHome(){ showTab('home'); loadHome(); }
function gotoClean(){ showTab('clean'); showView('scan'); }
function gotoSettings(){ showTab('settings'); showView('settings'); loadCatSwitches(); loadTasks(); }
function goView(v){
  showTab('settings'); showView(v);
  if(v==='organize'){ loadCatSwitches(); }
  else if(v==='ai'){ loadAI(); }
  else if(v==='tasks'){ loadTasks(); }
  else if(v==='log'){ loadLog(); }
  else if(v==='whitelist'){ loadWhitelist(); }
}
function showView(v){
  document.querySelectorAll('.view').forEach(x=>x.classList.add('hidden'));
  const el=$('view-'+v); if(el) el.classList.remove('hidden');
}
/* ===== 首页 ===== */
async function loadHome(){
  const s=await api('/api/status');
  const live=$('live-txt');
  if(s&&s.daemon){ live.textContent='守护中'; live.classList.add('on');
    const sc=await api('/api/scan');
    lastScan=sc;
    if(sc&&sc.ts){
      let tot=0; for(const c of Object.keys(sc.cats||{})) tot+=(+sc.cats[c].kb||0);
      $('lastscan').textContent=T('last_scan')+' '+sc.ts+' · '+T('files')+' '+fmtKB(tot);
      const avail=sc.free_kb*1024;
      $('home-bar').style.width=Math.min(100,40+(sc.free_kb/1024/1024)*5)+'%';
      animNum($('home-free'),avail);
      // 数据卡
      let jc=0; for(const c of Object.keys(sc.cats||{})){ jc+=(+sc.cats[c].count||0); }
      $('st-junk').textContent=jc+'项';
      $('st-big').textContent=(sc.big||[]).length+'个';
      if(s.stats){ $('st-clean').textContent=fmtKB(s.stats.kb*1024); $('st-count').textContent=s.stats.del+'次'; }
      loadHistory(sc);
    } else { $('lastscan').textContent=T('no_scan'); animNum($('home-free'),(s.free_kb||0)*1024); }
  } else { live.textContent='守护离线'; live.classList.remove('on'); $('home-free').textContent='--'; }
}
async function loadHistory(sc){
  const r=await api('/api/stats-history'); const h=r&&r.history||[]; if(!h.length){ $('chart').innerHTML='<div class="muted" style="width:100%;text-align:center;padding:14px">暂无清理记录</div>'; return; }
  const days=[],now=new Date();
  for(let i=6;i>=0;i--){ const d=new Date(now); d.setDate(d.getDate()-i); days.push(d.toISOString().slice(0,10)); }
  const vals=days.map(d=>{ const sum=h.filter(x=>x.d===d).reduce((a,x)=>a+(+x.kb||0),0); return sum; });
  const max=Math.max(...vals,1);
  $('chart').innerHTML=vals.map((v,i)=>'<div class="col"><div class="b" style="height:'+Math.max(2,Math.round(v/max*44))+'px"></div><div class="d">'+days[i].slice(5)+'</div></div>').join('');
}
/* ===== 扫描 ===== */
let scanTimer=null,cleanTimer=null;
async function startScan(){
  showTab('clean'); showView('scan');
  $('scan-pct').textContent='0%'; $('scan-cur').textContent='准备中…';
  $('scan-arc').style.strokeDashoffset=415;
  await api('/api/scan',{method:'POST',body:'{"force":1}'});
  clearInterval(scanTimer);
  scanTimer=setInterval(pollScan,600);
}
async function pollScan(){
  const p=await api('/api/progress');
  if(p&&p.cur){
    const m=p.cur.match(/(\d+)%/); const pct=m?parseInt(m[1]):0;
    $('scan-pct').textContent=pct+'%';
    $('scan-arc').style.strokeDashoffset=415*(1-pct/100);
    $('scan-cur').textContent=(p.msg||'扫描中…').replace(/^PROG\s*\d+\s*/,'');
  }
  const sc=await api('/api/scan');
  if(sc&&sc.ts&&(!lastScan||lastScan.ts!==sc.ts)){
    clearInterval(scanTimer); lastScan=sc; renderReview();
  }
}
/* ===== 清单 ===== */
let scan=null,lastScan=null,sel={},rulesStore={};
function renderReview(){
  scan=lastScan; sel={};
  if(!window.big_min){ api('/api/config').then(cfg=>{ const m=(cfg&&cfg.cfg||'').match(/big_min=(\d+)/); window.big_min=m?Math.round(parseInt(m[1])/1024):100; }); }
  const box=$('catlist'); box.innerHTML='';
  CAT_ORDER.forEach(id=>{
    const v=scan.cats&&scan.cats[id]; if(!v||!(+v.kb||+v.count)) return;
    const kb=+v.kb||0;
    const el=document.createElement('div');
    el.className='catcard'+(id==='social'?' redline':'');
    el.dataset.cat=id;
    const ico=(CATS[id]||[])[1]||'•';
    el.innerHTML='<div class="chk">✓</div><div class="catico">'+ico+'</div>'+
      '<div class="nm">'+(CATS[id]||[id])[0]+'<div class="cnt">'+v.count+' '+T('files')+(v.old?' · '+T('days_ago')+' '+fmtKB(v.old*1024):'')+'</div></div>'+
      '<div class="sz">'+fmtKB(kb)+'</div>';
    el.onclick=()=>{ sel[id]=!sel[id]; el.classList.toggle('on',!!sel[id]); updateSel(); };
    box.appendChild(el);
    renderRules(id,el);
  });
  renderBig();
  renderApps();
  showTab('clean'); showView('review');
  updateSel();
}
function updateSel(){
  let n=0,sum=0;
  for(const c of Object.keys(sel)){ if(!sel[c]) continue; n++;
    if(scan.cats&&scan.cats[c]) sum+=(+scan.cats[c].kb||0);
    if(c==='big'&&scan.big) scan.big.forEach(b=>sum+=(+b.kb||0));
  }
  $('sel-count').textContent=n; $('sel-size').textContent=fmtKB(sum);
  $('btn-clean').disabled=n===0;
}
function renderBig(){
  const big=scan.big||[]; if(!big.length) return;
  let total=0; big.forEach(b=>total+=(+b.kb||0));
  const el=document.createElement('div');
  el.className='catcard'+(sel.big?' on':'');
  el.dataset.cat='big';
  el.innerHTML='<div class="chk">✓</div><div class="catico">📈</div>'+
    '<div class="nm">大文件<div class="cnt">'+big.length+' '+T('count')+' · ≥'+(window.big_min||100)+'MB</div></div>'+
    '<div class="sz">'+fmtKB(total)+'</div>';
  el.onclick=()=>{ sel.big=!sel.big; el.classList.toggle('on',!!sel.big); updateSel(); };
  $('catlist').appendChild(el);
  const detail=document.createElement('div'); detail.className='hidden';
  detail.style.cssText='padding:2px 16px 12px;font-size:12px;color:var(--sub)';
  const extSet=[...new Set(big.map(b=>(b.p.match(/\.(\w+)$/)||['',''])[1]).filter(Boolean))];
  detail.innerHTML='<div style="display:flex;gap:6px;margin-bottom:6px;flex-wrap:wrap">'+
    '<span class="sw on" style="padding:2px 8px" data-ext="">所有</span>'+
    extSet.map(e=>'<span class="sw" style="padding:2px 8px" data-ext="'+e+'">.'+e+'</span>').join('')+'</div>'+
    big.slice(0,15).map(b=>'<div class="bigitem" data-ext="'+((b.p.match(/\.(\w+)$/)||['',''])[1])+'" style="display:flex;justify-content:space-between;gap:6px;padding:3px 0"><span style="word-break:break-all">'+fmtKB(b.kb)+' · '+escapeHtml(b.p)+'</span><span style="flex-shrink:0"><button class="sw" style="padding:1px 6px;font-size:10px" onclick="bigArchive(\''+escapeHtml(b.p)+'\')">归档</button> <button class="sw" style="padding:1px 6px;font-size:10px;color:var(--danger)" onclick="bigDelete(\''+escapeHtml(b.p)+'\')">删除</button></span></div>').join('')+
    (big.length>15?'<div style="padding:3px 0;color:var(--sub)">…还有 '+ (big.length-15)+' 个</div>':'');
  el.appendChild(detail);
  el.querySelector('.nm').style.cursor='pointer';
  el.querySelector('.nm').onclick=e=>{ e.stopPropagation(); detail.classList.toggle('hidden'); };
  detail.querySelectorAll('.sw[data-ext]').forEach(sw=>sw.onclick=()=>{
    detail.querySelectorAll('.sw[data-ext]').forEach(x=>x.classList.remove('on'));
    sw.classList.add('on');
    detail.querySelectorAll('.bigitem').forEach(it=>{ it.style.display=!sw.dataset.ext||it.dataset.ext===sw.dataset.ext?'':'none'; });
  });
}
/* ===== 规则路径行 ===== */
async function renderRules(catId,card){
  if(['big','empty','zero','temp','uninst','sqlite'].includes(catId)) return;
  const r=await api('/api/rules?type='+catId); if(!r||!r.content) return;
  const lines=r.content.split('\n').map(l=>l.trimEnd());
  const paths=lines.map(l=>{ const t=l.trim(); if(!t||t.startsWith('#')||t.startsWith('@')) return null; const p=t.split(/\s+/)[0]; return p.startsWith('/')?p:null; }).filter(Boolean);
  let exists={};
  if(paths.length){ const chk=await api('/api/check',{method:'POST',body:JSON.stringify({paths})}); exists=(chk&&chk.exists)||{}; }
  const box=document.createElement('div'); box.className='rulelist';
  lines.forEach((l,idx)=>{
    const t=l.trim(); if(!t||t.startsWith('#')||t.startsWith('@')) return;
    const parts=t.split(/\s+/); const path=parts[0]; if(!path.startsWith('/')) return;
    const flags=parts.slice(1);
    const row=document.createElement('div'); row.className='ruleline';
    row.innerHTML='<div class="rulepath">'+escapeHtml(path)+(exists[path]?'':'<span class="rulemiss">⚠️ 不存在</span>')+
      (flags.includes('high')?'<span class="rulemiss">⚠️ 高风险</span>':'')+'</div>'+
      '<div class="rulesw"><button class="sw '+(flags.includes('recurse')?'on':'')+'" data-k="recurse" data-i="'+idx+'">子目录</button>'+
      '<button class="sw '+(flags.includes('no-integrity')?'':'on')+'" data-k="integrity" data-i="'+idx+'">完整性</button></div>';
    box.appendChild(row);
    row.querySelector('[data-k=recurse]').onclick=()=>toggleRuleFlag(catId,idx,'recurse',flags,r.content,row);
    row.querySelector('[data-k=integrity]').onclick=()=>toggleRuleFlag(catId,idx,'no-integrity',flags,r.content,row);
  });
  card.appendChild(box);
}
async function toggleRuleFlag(cat,idx,flag,cur,content,row){
  const parts=content.split('\n'); let line=parts[idx]||'';
  if(flag==='recurse'){ if(cur.includes('recurse')) line=line.replace(/\s*recurse/,''); else line+=' recurse'; }
  else { if(cur.includes('no-integrity')) line=line.replace(/\s*no-integrity/,''); else line+=' no-integrity'; }
  parts[idx]=line;
  const rr=await api('/api/rules?type='+cat,{method:'POST',body:parts.join('\n')});
  if(rr.ok){ toast('已保存'); row.querySelector('[data-k="'+flag+'"]').classList.toggle('on'); }
}
/* ===== 清理 ===== */
async function doClean(){
  const items=Object.keys(sel).filter(c=>sel[c]).join(',');
  showTab('clean'); showView('cleaning');
  $('clean-pct').textContent='0%'; $('clean-arc').style.strokeDashoffset=415;
  await api('/api/clean',{method:'POST',body:JSON.stringify({cats:items})});
  clearInterval(cleanTimer); cleanTimer=setInterval(pollClean,600);
}
async function pollClean(){
  const p=await api('/api/progress');
  if(p&&p.cur){ const m=p.cur.match(/(\d+)%/); const pct=m?parseInt(m[1]):0;
    $('clean-pct').textContent=pct+'%';
    $('clean-arc').style.strokeDashoffset=415*(1-pct/100);
    $('clean-cur').textContent=(p.msg||'清理中…').replace(/^PROG\s*\d+\s*/,'');
  }
  const st=await api('/api/status');
  if(st&&!st.busy){ clearInterval(cleanTimer); finishClean(); }
}
async function finishClean(){
  const sc=await api('/api/scan');
  let freed=0; for(const c of Object.keys(sel)){ if(sc.cats&&sc.cats[c]) freed+=(+sc.cats[c].kb||0); }
  $('done-size').textContent=fmtKB(freed);
  $('done-msg').textContent=T('cleaned_done')+' '+fmtKB(freed);
  showView('done');
  loadHome(); loadCatSwitches();
}
/* ===== 设置 ===== */
let cfg={};
async function loadCatSwitches(){
  const r=await api('/api/config');
  cfg={}; if(r&&r.cfg){ r.cfg.split('\n').forEach(l=>{const i=l.indexOf('='); if(i>0) cfg[l.slice(0,i)]=l.slice(i+1);}); }
  const box=$('cat-switches'); box.innerHTML='';
  CAT_ORDER.forEach(id=>{
    const on=cfg['cat_'+id]!=='0';
    const row=document.createElement('div'); row.className='setrow';
    row.innerHTML='<span class="lbl">'+(CATS[id]||[id])[0]+'</span><div class="switch '+(on?'on':'')+'" data-cat="'+id+'"></div>';
    row.querySelector('.switch').onclick=async e=>{ e.stopPropagation();
      const s=e.target; s.classList.toggle('on'); cfg['cat_'+id]=s.classList.contains('on')?'1':'0'; await saveConfig(); };
    box.appendChild(row);
  });
  const trash=cfg.trash!=='0';
  $('sw-trash').classList.toggle('on',trash);
  $('sw-trash').onclick=async e=>{ e.stopPropagation();
    const s=e.target; s.classList.toggle('on'); cfg.trash=s.classList.contains('on')?'1':'0'; await saveConfig(); };
}
async function saveConfig(){
  const lines=Object.keys(cfg).map(k=>k+'='+cfg[k]);
  const r=await api('/api/config',{method:'POST',body:lines.join('\n')+'\n'});
  if(r.ok) toast('已保存');
}
function setTheme(t){ document.documentElement.dataset.theme=t; localStorage.setItem('jc_theme',t); }
function setLang(l){ LANG=l; localStorage.setItem('jc_lang',l);
  document.querySelectorAll('[data-i18n]').forEach(el=>{ const k=el.dataset.i18n; if(k) el.textContent=T(k); });
  document.querySelectorAll('.tab span').forEach((el,i)=>el.textContent=[T('nav_home'),T('nav_clean'),T('nav_settings')][i]);
  $('sel-lang')&&($('sel-lang').value=LANG);
  // 重建 CATS 分类名（const 对象不可替换但属性可改）
  Object.keys(CATS).forEach(k=>{ CATS[k][0]=T('cat_'+k); });
  gotoHome(); loadCatSwitches(); }
/* ===== 工具 ===== */
async function postRun(ep){
  const msgs={classify:T('classifying'),duplicate:T('deduping'),fstrim:T('maintaining'),rescan:T('refreshing')};
  toast(msgs[ep]||'执行中…');
  const r=await api('/api/'+ep,{method:'POST',body:'{}'});
  toast(r&&r.ok?'✓ '+T('done'):'✗ '+(r&&r.e||T('failed')));
  if(ep==='classify'||ep==='duplicate') loadHome();
}
async function classifyPreview(){
  toast('生成预览…');
  await api('/api/classify',{method:'POST',body:'{"preview":1}'});
  await new Promise(r=>setTimeout(r,900));
  const res=await api('/api/classify-preview');
  const files=res&&res.files||[];
  if(!files.length){ toast('预览为空'); return; }
  let html='<div style="max-height:50vh;overflow-y:auto;font-size:12px">';
  files.forEach(f=>{ html+='<div style="padding:4px 0;border-bottom:1px solid var(--line);word-break:break-all">'+escapeHtml(f.s)+' → '+escapeHtml(f.d)+'</div>'; });
  html+='</div>'+'<div style="margin-top:10px;font-size:12px;color:var(--sub)">共 '+files.length+' '+T('files')+'</div>';
  showDialog('📂 分类预览',html);
}
let dupKeep=null;
async function dupPreview(){
  toast('扫描重复文件中…');
  await api('/api/duplicate',{method:'POST',body:'{"preview":1}'});
  await new Promise(r=>setTimeout(r,1000));
  const res=await api('/api/duplicate-preview');
  const files=res&&res.files||[];
  if(!files.length){ toast('未发现重复文件'); return; }
  dupKeep=files.find(f=>f.keep)?files.find(f=>f.keep).p:files[0].p;
  let html='<div style="max-height:45vh;overflow-y:auto;font-size:12px">';
  files.forEach(f=>{ const isKeep=f.p===dupKeep;
    html+='<div style="padding:4px 0;border-bottom:1px solid var(--line);word-break:break-all;display:flex;justify-content:space-between;gap:6px">'+
      (isKeep?'<span style="color:var(--ok);font-weight:600">✔ 保留</span>':'<span style="color:var(--danger)">将处理</span>')+' · '+escapeHtml(f.p)+
      (isKeep?'':'<button class="sw" style="padding:2px 6px" data-keep="'+escapeHtml(f.p)+'">设为保留</button>')+'</div>'; });
  html+='</div><div style="margin-top:8px;display:flex;gap:6px;flex-wrap:wrap">'+
    '<button class="sw on" onclick="dupAction(\'move\')">默认归档</button>'+
    '<button class="sw" onclick="dupAction(\'del\')">默认删除</button>'+
    '<button class="sw" onclick="dupAction(\'keep\')">✔保留归档</button>'+
    '<button class="sw" onclick="dupAction(\'keepdel\')">✔保留删除</button></div>';
  showDialog('📑 重复文件',html);
  document.querySelectorAll('[data-keep]').forEach(b=>b.onclick=()=>{ dupKeep=b.dataset.keep; toast('已指定保留'); });
}
async function dupAction(mode){
  let body={};
  if(mode==='del') body={delete:1};
  else if(mode==='keep') body={keep:dupKeep};
  else if(mode==='keepdel') body={delete:1,keep:dupKeep};
  await api('/api/duplicate',{method:'POST',body:JSON.stringify(body)});
  toast(mode.includes('del')?'✔ 已删除':'✔ 已归档');
  const d=document.querySelector('div[style]'); if(d) d.remove();
}
function dupDeleteConfirm(){ showDialog('📑 删除重复','<div style="font-size:13px;color:var(--sub)">删除重复文件副本（保留原始），确认？</div><div style="margin-top:8px;text-align:center"><button class="sw on" onclick="dupDelete()">确认删除</button></div>'); }
async function dupDelete(){ await api('/api/duplicate',{method:'POST',body:'{"delete":1}'}); toast('✔ 已删除'); const d=document.querySelector('div[style]'); if(d) d.remove(); }
/* ===== AI ===== */
let aiHist=[],aiLoading=false;
async function loadAI(){
  const r=await api('/api/config');
  const lines=(r&&r.cfg||'').split('\n');
  lines.forEach(l=>{ const i=l.indexOf('='); if(i>0&&l.startsWith('ai_')) cfg[l.slice(0,i)]=l.slice(i+1); });
  if($('ai-base')) $('ai-base').value=cfg.ai_base||'';
  if($('ai-key')) $('ai-key').value=cfg.ai_key||'';
  if($('ai-model')) $('ai-model').value=cfg.ai_model||'';
}
async function saveAI(){
  const lines=(await api('/api/config')).cfg.split('\n').filter(l=>l&&!l.startsWith('ai_'));
  lines.push('ai_base='+$('ai-base').value,'ai_key='+$('ai-key').value,'ai_model='+$('ai-model').value);
  const rr=await api('/api/config',{method:'POST',body:lines.join('\n')+'\n'});
  toast(rr.ok?'已保存':'✗');
}
function addAIMsg(role,text){
  const chat=$('ai-chat'); if(!chat) return;
  const div=document.createElement('div');
  div.style.cssText='padding:8px 11px;margin:5px 0;border-radius:14px;max-width:88%;font-size:13px;line-height:1.5;word-break:break-all;'+
    (role==='user'?'align-self:flex-end;margin-left:auto;background:linear-gradient(135deg,#00e5c8,#7c5cff);color:#02100e;border-bottom-right-radius:4px;box-shadow:0 3px 12px rgba(0,229,200,.25)':
    'align-self:flex-start;background:linear-gradient(180deg,rgba(255,255,255,.08),rgba(255,255,255,.03));border:1px solid var(--line);border-bottom-left-radius:4px');
  div.textContent=text;
  chat.style.display='flex'; chat.style.flexDirection='column'; chat.style.alignItems='flex-start';
  chat.appendChild(div); chat.scrollTop=chat.scrollHeight;
}
async function doAI(){
  if(aiLoading) return;
  const input=$('ai-input'); const q=input.value.trim(); if(!q) return;
  input.value=''; aiLoading=true;
  addAIMsg('user',q); addAIMsg('assistant','思考中…');
  const r=await api('/api/ai',{method:'POST',body:JSON.stringify({q,history:aiHist})});
  aiHist.push({role:'user',content:q});
  const txt=r&&r.ai||(r.e||'AI 错误');
  const chat=$('ai-chat'); chat.removeChild(chat.lastChild);
  addAIMsg('assistant',txt); aiHist.push({role:'assistant',content:txt});
  aiLoading=false;
  const names=[['cache','缓存'],['junk','系统垃圾'],['apk','安装包'],['zip','压缩包'],['thumb','缩略图'],['log','日志'],['zero','空文件']];
  const matched=names.filter(n=>txt.match(new RegExp(n[1],'i'))).map(n=>n[0]);
  if(matched.length){
    const wrap=document.createElement('div'); wrap.style.cssText='padding:4px 8px;text-align:center';
    wrap.innerHTML='<button class="sw on" style="padding:4px 10px">✅ 采纳：'+matched.map(c=>(CATS[c]||[c])[0]).join('、')+'</button>';
    chat.appendChild(wrap);
    wrap.querySelector('button').onclick=()=>{ matched.forEach(c=>{ sel[c]=1; }); showTab('clean'); renderReview(); toast('已采纳'); };
  }
}
/* ===== 定时 ===== */
let tasks=[];
async function loadTasks(){
  const r=await api('/api/tasks'); tasks=((r&&r.tasks)||'').split('\n').filter(l=>l.trim()); renderTasks();
}
function renderTasks(){
  const box=$('task-list'); if(!box) return;
  box.innerHTML='';
  if(!tasks.length){ box.innerHTML='<div class="empty"><div class="eico">⏰</div>暂无定时任务<br><span style="font-size:11px">在下方添加</span></div>'; return; }
  const now=new Date();
  const nextTimes=tasks.map(l=>{ const m=l.match(/daily=([0-9:]+)/); if(!m) return null;
    const hmi=m[1].split(':').map(Number); const n=new Date(now); n.setHours(hmi[0],hmi[1],0,0); if(n<=now) n.setDate(n.getDate()+1); return n; }).filter(Boolean);
  if(nextTimes.length){ const nt=nextTimes.reduce((a,b)=>a<b?a:b);
    const diff=Math.round((nt-now)/3600000*10)/10;
    box.innerHTML+='<div style="padding:8px;font-size:12px;color:var(--acc)">⏱ '+T('next_clean')+'：'+nt.toLocaleString()+'（'+diff+''+T('hours_after')+'）</div>';
  }
  tasks.forEach((l,i)=>{
    const enable=l.includes('enable=1');
    const every=l.match(/every=([^,]+)/), daily=l.match(/daily=([^,]+)/), cats=l.match(/cats=([^,]+)/);
    const cond=[l.includes('charge=1')?'充电':'',l.includes('wifi=1')?'WiFi':'',l.includes('idle=1')?'空闲':''].filter(Boolean).join('/');
    const row=document.createElement('div'); row.className='setrow';
    row.innerHTML='<div class="switch '+(enable?'on':'')+'" data-i="'+i+'"></div>'+
      '<span class="lbl" style="font-size:12px">'+(every?T('every')+' '+every[1]+' '+T('hours'):(daily?T('daily')+' '+daily[1]:''))+' · '+(cats?cats[1].replace(/,/g,'/'):'')+(cond?' <span style="color:var(--acc)">['+cond+']</span>':'')+'</span>'
      '<button class="sw" onclick="delTask('+i+')">🗑</button>';
    row.querySelector('.switch').onclick=()=>toggleTask(i);
    box.appendChild(row);
  });
}
async function toggleTask(i){
  const l=tasks[i];
  tasks[i]=l.includes('enable=1')?l.replace('enable=1','enable=0'):l.replace('enable=0','enable=1');
  await saveTasks();
}
async function delTask(i){ tasks.splice(i,1); await saveTasks(); }
async function saveTasks(){
  const r=await api('/api/tasks',{method:'POST',body:tasks.join('\n')+'\n'});
  toast(r.ok?'已保存':'✗'); renderTasks();
}
function toggleTaskMode(){ const m=$('task-mode').value; $('row-every').classList.toggle('hidden',m!=='every'); $('row-daily').classList.toggle('hidden',m!=='daily'); }
async function addTask(){
  const cats=$('task-cats').value; const m=$('task-mode').value;
  let line=m==='every'?('enable=1,every='+$('task-hours').value+'h,cats='+cats):('enable=1,daily='+$('task-daily').value+',cats='+cats);
  ['charge','wifi','idle'].forEach(id=>{ const el=document.getElementById('task-'+id); if(el&&el.checked) line+=','+id+'=1'; });
  tasks.push(line); await saveTasks();
}
/* ===== 规则 ===== */
async function loadRule(tp){ const r=await api('/api/rules?type='+tp); $('rule-text').value=(r&&r.content)||''; }
async function saveRule(){ const tp=$('sel-rule').value;
  const r=await api('/api/rules?type='+tp,{method:'POST',body:$('rule-text').value}); toast(r.ok?'已保存':'✗'); }
async function loadWhitelist(){ const r=await api('/api/rules?type=whitelist'); $('wl-text').value=(r&&r.content)||''; }
async function saveWhitelist(){
  const r=await api('/api/rules?type=whitelist',{method:'POST',body:$('wl-text').value}); toast(r.ok?'已保存':'✗'); }
const RULE_TEMPLATES={
  '精简':{'cache':['/sdcard/Android/data/*/cache'],'junk':['/sdcard/Download/*.tmp'],'apk':['/sdcard/Download/*.apk'],'zip':['/sdcard/Download/*.zip']},
  '标准':{'cache':['/sdcard/Android/data/*/cache','/data/data/*/cache'],'junk':['/sdcard/Download/*.tmp','/sdcard/Download/*.bak','/sdcard/MIUI/log'],'apk':['/sdcard/Download/*.apk'],'zip':['/sdcard/Download/*.zip']},
  '激进':{'cache':['/sdcard/Android/data/*/cache','/data/data/*/cache','/sdcard/Android/obb/*'],'junk':['/sdcard/Download/*.tmp','/sdcard/Download/*.bak','/sdcard/Pictures/.thumbnails'],'apk':['/sdcard/Download/*.apk','/sdcard/tmp/*.apk'],'zip':['/sdcard/Download/*.zip','/sdcard/Download/*.rar']}};
function showRuleTemplates(){
  let html=Object.keys(RULE_TEMPLATES).map(k=>'<div style="padding:8px;margin:4px 0;border-radius:8px;background:var(--card2);cursor:pointer" onclick="applyRuleTemplate(\''+k+'\')">'+k+'：'+
    Object.keys(RULE_TEMPLATES[k]).map(c=>c+':'+RULE_TEMPLATES[k][c].length+'条').join('、')+'</div>').join('');
  showDialog('📦 规则模板',html+'<div style="font-size:12px;color:var(--sub);margin-top:8px">点击套用（覆盖当前规则）</div>');
}
async function applyRuleTemplate(name){
  const t=RULE_TEMPLATES[name]; if(!t) return;
  for(const [type,paths] of Object.entries(t)){
    await api('/api/rules?type='+type,{method:'POST',body:paths.map(p=>'# '+type+'\n'+p).join('\n')});
  }
  toast('✔ 已应用「'+name+'」模板');
}
async function exportRules(){
  const types=['cache','junk','apk','zip','thumb','log','social','whitelist','classify'];
  const data={};
  for(const t of types){ const r=await api('/api/rules?type='+t); if(r&&r.content) data[t]=r.content; }
  const cfg2=await api('/api/config'); if(cfg2&&cfg2.cfg) data.config=cfg2.cfg;
  const txt=JSON.stringify(data,null,2);
  if(navigator.clipboard) navigator.clipboard.writeText(txt).then(()=>toast('已复制'));
  else prompt('复制 JSON：',txt);
  toast('✔ 已导出 '+Object.keys(data).length+' 项');
}
function importRules(){ showDialog('📥 导入规则','<textarea id="import-json" style="width:100%;height:200px;font-family:monospace;font-size:12px"></textarea><div style="margin-top:8px;text-align:center"><button class="sw on" onclick="doImport()">导入</button></div>'); }
async function doImport(){
  try{
    const data=JSON.parse($('import-json').value);
    for(const [type,content] of Object.entries(data)){
      if(type==='config'){ await api('/api/config',{method:'POST',body:content}); continue; }
      await api('/api/rules?type='+type,{method:'POST',body:content});
    }
    toast('✔ 导入 '+Object.keys(data).length+' 项'); const d=$('import-json').closest('div[style]'); if(d) d.remove();
  }catch(e){ toast('✗ JSON 格式错误'); }
}
/* ===== 日志 ===== */
async function loadLog(kind){
  const r=await api('/api/log'+(kind?('?type='+kind):''));
  if($('log-box')) $('log-box').textContent=(r&&r.log)||'(空)';
}
/* ===== 初始化 ===== */
function init(){
  const th=localStorage.getItem('jc_theme'); if(th) setTheme(th);
  loadHome();
  const ai=$('ai-input'); if(ai) ai.addEventListener('keydown',e=>{ if(e.key==='Enter') doAI(); });
}
function applyI18N(){ document.querySelectorAll('[data-i18n]').forEach(el=>{const k=el.dataset.i18n;if(k)el.textContent=T(k);}); document.querySelectorAll('.tab span').forEach((el,i)=>el.textContent=[T('nav_home'),T('nav_clean'),T('nav_settings')][i]); $('sel-lang')&&($('sel-lang').value=LANG); }
document.addEventListener('DOMContentLoaded',()=>{ applyI18N(); init(); });
