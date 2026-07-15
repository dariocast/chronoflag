import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import ClockCard from './ClockCard.svelte';

describe('ClockCard',()=>{it('shows lap only for stopwatch controllers',()=>{render(ClockCard,{props:{clock:{id:'c',type:'stopwatch',state:'idle',order:0,duration:0,accumulated:0,version:0,laps:[]},control:true,now:new Date()}});expect(screen.getByRole('button',{name:'Start'})).toBeInTheDocument();expect(screen.getByRole('button',{name:'Lap'})).toBeInTheDocument()});it('renders public timer without controls',()=>{render(ClockCard,{props:{clock:{id:'t',type:'timer',state:'idle',order:0,duration:60_000_000_000,accumulated:0,version:0,laps:[]},control:false,now:new Date()}});expect(screen.getByText('01:00')).toBeInTheDocument();expect(screen.queryByRole('button')).not.toBeInTheDocument()})});
