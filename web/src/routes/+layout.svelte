<script lang="ts">
    import '../app.css';
    import { onMount } from 'svelte';
    import { connectWS } from '$lib/websocket';

    let { children } = $props();

    onMount(() => {
        // Automatically reconnect WS proxy pointing to Go SIEM loop
        connectWS("ws://127.0.0.1:8080/ws");
    });
</script>

<div class="app-layout">
    {@render children()}
</div>
