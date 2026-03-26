<script lang="ts">
    import { networkEvents, isConnected } from '$lib/websocket';
    import { fade, slide } from 'svelte/transition';
    import ThreatBanner from '$lib/ThreatBanner.svelte';
    import TopologyGraph from '$lib/TopologyGraph.svelte';

    function getEventColor(type: string) {
        switch(type) {
            case 'THREAT_ALERT': return '#ff3333';
            case 'PORT_SCAN': return '#ff9900';
            case 'DNS': return '#00ffff';
            case 'ARP': return '#33ff33';
            default: return '#cccccc';
        }
    }
    
    function getEventBackground(type: string) {
        if(type === 'THREAT_ALERT') return 'rgba(255, 51, 51, 0.15)';
        return 'rgba(0,0,0,0.5)';
    }
</script>

<svelte:head>
    <title>O.W.A.S.A.K.A. Command Center</title>
</svelte:head>

<ThreatBanner />

<main class="container">
    <!-- Header Controls -->
    <header class="glass-panel header-panel">
        <div>
            <h1>O.W.A.S.A.K.A.</h1>
            <p class="text-muted">Air-gapped Command Center</p>
        </div>
        <div class="status-badge {$isConnected ? 'online' : 'offline'}">
            <span class="status-dot"></span>
            {$isConnected ? 'SYSTEM SECURED' : 'DISCONNECTED'}
        </div>
    </header>

    <!-- Main Live Streams -->
    <section class="dashboard-grid">
        <div class="glass-panel feed-panel">
            <h2>Live Intelligence Feed</h2>
            <div class="event-list">
                {#each $networkEvents as event (event.id || Math.random())}
                    <div class="event-card animate-enter" style="background: {getEventBackground(event.type)}" transition:slide>
                        <div class="event-header">
                            <span style="color: {getEventColor(event.type)}; font-weight: bold;">{event.type || 'SYSTEM'}</span>
                            <span class="text-muted">{new Date(event.timestamp || Date.now()).toLocaleTimeString()}</span>
                        </div>
                        <pre>{JSON.stringify(event.metadata || event, null, 2)}</pre>
                    </div>
                {/each}
                
                {#if $networkEvents.length === 0}
                    <div class="empty-state text-muted" transition:fade>
                        <div style="font-size: 2rem; margin-bottom: 1rem; opacity: 0.5;">📡</div>
                        Awaiting network telemetry...
                    </div>
                {/if}
            </div>
        </div>
        
        <!-- Node Topology Graph Visualization -->
        <div class="glass-panel stats-panel">
            <h2>Network Topology</h2>
            <div style="margin-top: 1rem; width: 100%; height: 100%;">
                <TopologyGraph />
            </div>
        </div>
    </section>
</main>

<style>
    .container {
        max-width: 1400px;
        margin: 0 auto;
        padding: 2rem;
        display: flex;
        flex-direction: column;
        gap: 2rem;
    }

    .header-panel {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 2rem;
    }

    .dashboard-grid {
        display: grid;
        grid-template-columns: 2fr 1fr;
        gap: 2rem;
        align-items: start;
    }

    .feed-panel, .stats-panel {
        min-height: 60vh;
    }

    .event-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        max-height: calc(80vh - 100px);
        overflow-y: auto;
        padding-right: 0.5rem;
    }

    .event-card {
        background: rgba(0,0,0,0.5);
        border: 1px solid rgba(255,255,255,0.02);
        border-radius: 8px;
        padding: 1.25rem;
        transition: transform 0.2s, background 0.2s;
    }
    
    .event-card:hover {
        background: rgba(0,0,0,0.7);
        border-color: rgba(255,255,255,0.1);
    }

    .event-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.75rem;
        font-family: 'Fira Code', monospace;
        font-size: 0.85rem;
        font-weight: 600;
        letter-spacing: 0.05em;
    }

    pre {
        color: var(--fg-muted);
        font-family: 'Fira Code', monospace;
        font-size: 0.8rem;
        overflow-x: auto;
        white-space: pre-wrap;
    }

    .empty-state {
        text-align: center;
        padding: 4rem 2rem;
        border: 1px dashed rgba(255,255,255,0.1);
        border-radius: 12px;
        background: rgba(0,0,0,0.2);
    }
    
    /* Custom Scrollbar for the feed */
    .event-list::-webkit-scrollbar {
        width: 6px;
    }
    .event-list::-webkit-scrollbar-track {
        background: rgba(255, 255, 255, 0.02);
        border-radius: 10px;
    }
    .event-list::-webkit-scrollbar-thumb {
        background: rgba(255, 255, 255, 0.1);
        border-radius: 10px;
    }
    .event-list::-webkit-scrollbar-thumb:hover {
        background: rgba(255, 255, 255, 0.2);
    }
</style>
