<script lang="ts">
  import type { Clock } from './types';
  import { displayTime, detailTime } from './clock';
  export let clock:Clock; export let control=false; export let now=new Date(); export let highlighted=false;
  export let onaction:(type:string)=>void=()=>{}; export let onlabel:(label:string)=>void=()=>{}; export let onhighlight:()=>void=()=>{}; export let onremove:()=>void=()=>{};
  $: primary=clock.state==='idle'?'Start':clock.state==='running'?'Pause':clock.state==='paused'?'Resume':'Reset';
  $: primaryCommand=primary.toLowerCase();
</script>
<article class:hero={highlighted} class="clock-card" data-state={clock.state}>
  <header><span class="kind">{clock.type==='timer'?'TIMER':'STOPWATCH'}</span><strong>{clock.state.toUpperCase()}</strong></header>
  {#if control}<input class="label" aria-label="Clock label" maxlength="80" value={clock.label??''} placeholder="UNTITLED" on:change={(e)=>onlabel(e.currentTarget.value)} />{:else if clock.label}<h2>{clock.label}</h2>{/if}
  <div class="time" aria-live="off">{displayTime(clock,now)}</div>
  {#if clock.state!=='running'&&clock.type==='stopwatch'}<div class="detail">{detailTime(clock.accumulated)}</div>{/if}
  {#if control}
    <div class="controls"><button class="primary" on:click={()=>onaction(primaryCommand)}>{primary}</button>{#if clock.type==='stopwatch'}<button disabled={clock.state!=='running'} on:click={()=>onaction('lap')}>Lap</button>{/if}<button class="quiet" on:click={()=>onaction('reset')}>Reset</button></div>
    <div class="tools"><button on:click={onhighlight}>{highlighted?'Unfocus':'Focus'}</button><button class="danger" on:click={onremove}>Delete</button></div>
  {/if}
  {#if clock.type==='stopwatch'&&clock.laps?.length}<ol class="laps">{#each [...clock.laps].reverse() as lap}<li><b>#{lap.number}</b><span>{detailTime(lap.elapsed)}</span><small>+{detailTime(lap.split)}</small></li>{/each}</ol>{/if}
</article>
