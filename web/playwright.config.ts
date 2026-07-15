import { defineConfig } from '@playwright/test';
export default defineConfig({testDir:'./e2e',use:{baseURL:'http://127.0.0.1:18080'},webServer:{command:'cd .. && make build && LISTEN_ADDR=:18080 ./chronograph',url:'http://127.0.0.1:18080/healthz',reuseExistingServer:false,timeout:120000}});
