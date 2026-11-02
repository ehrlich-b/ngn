'use strict';
const worker = new Worker('worker.js');
const board = document.querySelector('#board'), status = document.querySelector('#status'), turn = document.querySelector('#turn'), movesEl = document.querySelector('#moves'), promotion = document.querySelector('#promotion');
const pieces = {K:'♔',Q:'♕',R:'♖',B:'♗',N:'♘',P:'♙',k:'♚',q:'♛',r:'♜',b:'♝',n:'♞',p:'♟'};
let moves=[], state=null, selected='', side='w', busy=true, pendingPromotion='', ready=false;
function request(extra={}) {busy=true; worker.postMessage({moves,...extra});}
function draw() {
 const [placement,active]=state.fen.split(' '), rows=placement.split('/'), cells={};
 rows.forEach((row,y)=>{let x=0; for(const c of row){if(/\d/.test(c)) x+=Number(c); else {cells['abcdefgh'[x]+(8-y)]=c; x++;}}});
 board.replaceChildren(); const order=side==='w'?[0,1,2,3,4,5,6,7]:[7,6,5,4,3,2,1,0];
 for(const y of order) for(const x of order){const sq='abcdefgh'[x]+(8-y), b=document.createElement('button'); b.className='square '+((x+y)%2?'dark':'light')+(selected===sq?' selected':''); b.setAttribute('aria-label',sq+(cells[sq]?' '+cells[sq]:'')); b.dataset.square=sq; b.textContent=pieces[cells[sq]]||''; if(x===order[0]||y===order[7]) {const c=document.createElement('span'); c.className='coord'; c.textContent=(x===order[0]?8-y:'')+(y===order[7]?'abcdefgh'[x]:''); b.append(c)} b.onclick=()=>choose(sq); board.append(b);}
 turn.textContent=active==='w'?'White to move':'Black to move';
 movesEl.replaceChildren(); for(let i=0;i<moves.length;i+=2){const li=document.createElement('li'); li.textContent=moves[i]+(moves[i+1]?'  '+moves[i+1]:''); movesEl.append(li)}
}
function choose(sq){if(busy||!state||state.status||state.fen.split(' ')[1]!==side)return; if(selected){const options=state.legal.filter(m=>m.slice(0,4)===selected+sq); if(options.length===1){play(options[0]);return} if(options.length>1){pendingPromotion=selected+sq;promotion.hidden=false;return}} selected=sq;draw()}
function play(m){promotion.hidden=true;pendingPromotion='';selected='';moves.push(m);status.textContent='Thinking…';request()}
promotion.onclick=e=>{if(e.target.dataset.piece&&pendingPromotion)play(pendingPromotion+e.target.dataset.piece)};
worker.onmessage=e=>{const data=e.data;if(data.ready){ready=true;request();return} if(data.error){busy=false;status.textContent=data.error;return} if(data.bestmove){moves.push(data.bestmove);request();return} state=data;draw();if(data.status){busy=false;status.textContent=data.status==='checkmate'?'Checkmate':`Draw: ${data.status}`;return} if(data.fen.split(' ')[1]!==side){status.textContent='NGN is thinking…'; const seconds=Math.min(30,Math.max(.1,Number(document.querySelector('#time').value)||1));request({millis:Math.round(seconds*1000)});return} busy=false;status.textContent='Your move';};
document.querySelector('#new').onclick=()=>{if(!ready||busy)return;moves=[];selected='';promotion.hidden=true;side=document.querySelector('#side').value;status.textContent='Starting…';request()};
