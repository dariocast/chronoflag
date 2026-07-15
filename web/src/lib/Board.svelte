<script lang="ts">
  import { onMount } from 'svelte'; import ClockCard from './ClockCard.svelte'; import type { Snapshot,ClockType } from './types'; import * as api from './api';
  export let token:string; export let mode:'control'|'view'; let snapshot:Snapshot|null=null;let status='connecting';let now=new Date();let error='';let share=false;let adding=false;
  onMount(()=>{api.getSnapshot(mode,token).then(v=>snapshot=v).catch(e=>error=e.message);const stop=api.events(mode,token,v=>snapshot=v,s=>status=s);const tick=setInterval(()=>now=new Date(),10);return()=>{stop();clearInterval(tick)}});
  async function act(id:string,type:string){try{status='sending';snapshot=type==='undo'?await api.undo(token,id):await api.command(token,id,type);status='synced'}catch(e){error=(e as Error).message;status='offline'}}
  async function add(type:ClockType,durationMS=0){snapshot=await api.addClock(token,type,durationMS);adding=false}
  async function patch(id:string,body:object){snapshot=await api.patchClock(token,id,body)}
  async function remove(id:string){if(confirm('Delete this clock?'))snapshot=await api.removeClock(token,id)}
  $: control=mode==='control'; $: publicURL=snapshot?`${location.origin}/v/${snapshot.id}`:'';
</script>
<svelte:head><title>{control?'Race Control':'Live Board'} — Chronograph</title></svelte:head>
<header class="topbar"><a href="/" class="brand">CHRONOGRAPH</a><span class="status" data-state={status}>{status.toUpperCase()}</span>{#if control}<button on:click={()=>share=!share}>Share</button><button class="add" on:click={()=>adding=!adding}>+ Add clock</button>{/if}</header>
{#if share&&snapshot}<aside class="panel"><h2>SHARE LINKS</h2><label>CONTROL<input readonly value={location.href}/></label><button on:click={()=>navigator.clipboard.writeText(location.href)}>Copy control</button><label>PUBLIC<input readonly value={publicURL}/></label><button on:click={()=>navigator.clipboard.writeText(publicURL)}>Copy public</button><p>Anyone with the public link can watch. Keep the control link secret.</p></aside>{/if}
{#if adding}<aside class="panel"><h2>ADD CLOCK</h2><button on:click={()=>add('stopwatch')}>Stopwatch</button><div class="presets">{#each [1,3,5,10,15,30,60] as min}<button on:click={()=>add('timer',min*60000)}>{min} min</button>{/each}</div></aside>{/if}
{#if error}<div class="error" role="alert">{error}<button on:click={()=>location.reload()}>Retry</button></div>{/if}
{#if snapshot}<main class:public-board={!control} class="board">{#each snapshot.clocks as c (c.id)}<ClockCard clock={c} {control} {now} highlighted={snapshot.highlighted_clock_id===c.id} onaction={(type)=>act(c.id,type)} onlabel={(label)=>patch(c.id,{label})} onhighlight={()=>patch(c.id,{highlighted:snapshot?.highlighted_clock_id!==c.id})} onremove={()=>remove(c.id)}/>{/each}</main>{:else}<main class="loading">CONNECTING…</main>{/if}
