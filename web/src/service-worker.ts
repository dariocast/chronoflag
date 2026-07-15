/// <reference lib="webworker" />
import { build, files, version } from '$service-worker';
const worker=self as unknown as ServiceWorkerGlobalScope;const cache=`chronograph-${version}`;const shell=[...build,...files];
worker.addEventListener('install',(event)=>event.waitUntil(caches.open(cache).then(c=>c.addAll(shell))));
worker.addEventListener('activate',(event)=>event.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k!==cache).map(k=>caches.delete(k))))));
worker.addEventListener('fetch',(event)=>{if(event.request.method!=='GET'||new URL(event.request.url).pathname.startsWith('/api/'))return;event.respondWith(caches.match(event.request).then(hit=>hit??fetch(event.request)))});
