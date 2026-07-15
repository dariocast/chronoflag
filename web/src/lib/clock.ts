import type { Clock } from './types';
const NS=1_000_000_000;
export function elapsedAt(clock:Clock,now=new Date()):number{let value=clock.accumulated;if(clock.state==='running'&&clock.anchor)value+=Math.max(0,now.getTime()-new Date(clock.anchor).getTime())*1_000_000;return clock.type==='timer'?Math.min(clock.duration,value):Math.max(0,value)}
function parts(seconds:number){return `${String(Math.floor(seconds/60)).padStart(2,'0')}:${String(Math.floor(seconds%60)).padStart(2,'0')}`}
export function displayTime(clock:Clock,now=new Date()):string{const elapsed=elapsedAt(clock,now);if(clock.type==='timer'){const left=Math.max(0,clock.duration-elapsed);return parts(Math.ceil(left/NS))}const cs=Math.floor(elapsed/10_000_000);return `${parts(Math.floor(cs/100))}.${String(cs%100).padStart(2,'0')}`}
export function detailTime(ns:number):string{const ms=Math.floor(Math.max(0,ns)/1_000_000);return `${parts(Math.floor(ms/1000))}.${String(ms%1000).padStart(3,'0')}`}
