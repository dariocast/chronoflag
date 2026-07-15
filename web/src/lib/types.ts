export type ClockType='stopwatch'|'timer';
export type ClockState='idle'|'running'|'paused'|'expired';
export interface Lap{number:number;elapsed:number;split:number;label?:string}
export interface Clock{id:string;type:ClockType;state:ClockState;label?:string;order:number;duration:number;accumulated:number;anchor?:string;version:number;laps:Lap[]}
export interface Snapshot{id:string;tier:'free'|'premium';lifecycle:'active'|'archived';clocks:Clock[];highlighted_clock_id?:string;version:number;last_control_at:string;server_time:string}
export interface Created{instance_id:string;control_url:string;view_url:string}
