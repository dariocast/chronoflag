import { describe, expect, it } from 'vitest';
import { displayTime, elapsedAt } from './clock';
import type { Clock } from './types';

const stopwatch: Clock={id:'c',type:'stopwatch',state:'running',order:0,duration:0,accumulated:2_000_000_000,anchor:'2026-07-14T12:00:00Z',version:1,laps:[]};
describe('clock projection',()=>{
  it('projects running stopwatch from server anchor',()=>{expect(elapsedAt(stopwatch,new Date('2026-07-14T12:00:01.347Z'))).toBe(3_347_000_000);expect(displayTime(stopwatch,new Date('2026-07-14T12:00:01.347Z'))).toBe('00:03.34')});
  it('ceil-displays timer seconds and clamps at zero',()=>{const timer={...stopwatch,type:'timer' as const,duration:5_000_000_000,accumulated:0};expect(displayTime(timer,new Date('2026-07-14T12:00:04.001Z'))).toBe('00:01');expect(displayTime(timer,new Date('2026-07-14T12:00:08Z'))).toBe('00:00')});
});
