import { writable } from 'svelte/store';

export const networkEvents = writable<any[]>([]);
export const isConnected = writable(false);

let ws: WebSocket | null = null;

export function connectWS(url: string = "ws://127.0.0.1:8080/ws") {
    if (ws && ws.readyState !== WebSocket.CLOSED) return;

    ws = new WebSocket(url);

    ws.onopen = () => {
        console.log("🟢 Connected to O.W.A.S.A.K.A Core");
        isConnected.set(true);
    };
    
    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            networkEvents.update(cur => [data, ...cur].slice(0, 500)); // hold max 500
        } catch(e) {
            console.error("Failed to parse event", e);
        }
    };
    
    ws.onclose = () => {
        console.log("🔴 Disconnected from core, retrying...");
        isConnected.set(false);
        setTimeout(() => connectWS(url), 2000);
    };
}
