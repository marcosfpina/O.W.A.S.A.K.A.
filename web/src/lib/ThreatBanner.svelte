<script lang="ts">
    import { networkEvents } from '$lib/websocket';
    import { slide } from 'svelte/transition';
    
    let activeAlert = $state<any>(null);
    let timeout: ReturnType<typeof setTimeout>;

    $effect(() => {
        const latest = $networkEvents[0];
        if (latest && latest.type === 'THREAT_ALERT') {
            activeAlert = latest;
            if (timeout) clearTimeout(timeout);
            timeout = setTimeout(() => {
                activeAlert = null; // Auto-dismiss after 6s
            }, 6000);
        }
    });

    function dismiss() {
        activeAlert = null;
        if (timeout) clearTimeout(timeout);
    }
</script>

{#if activeAlert}
<div class="threat-banner" transition:slide={{duration: 400}}>
    <div class="threat-content">
        <span class="icon">⚠️</span>
        <div class="message">
            <strong>CRITICAL THREAT DETECTED</strong>
            <p>{activeAlert.metadata?.reason || 'Unknown anomaly detected'}</p>
            <p class="target">Target: {activeAlert.metadata?.target || 'Unknown'}</p>
        </div>
    </div>
    <button class="dismiss" onclick={dismiss}>✕</button>
</div>
{/if}

<style>
    .threat-banner {
        position: fixed;
        bottom: 2rem;
        left: 50%;
        transform: translateX(-50%);
        z-index: 9999;
        background: rgba(220, 20, 60, 0.9);
        backdrop-filter: blur(10px);
        border: 1px solid rgba(255, 100, 100, 0.5);
        border-radius: 12px;
        padding: 1rem 2rem;
        color: white;
        box-shadow: 0 10px 40px rgba(220, 20, 60, 0.5);
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 90%;
        max-width: 700px;
    }
    .threat-content {
        display: flex;
        align-items: center;
        gap: 1.5rem;
    }
    .icon {
        font-size: 3rem;
        animation: pulseFade 1.5s infinite;
    }
    .message strong {
        font-size: 1.3rem;
        letter-spacing: 0.1em;
        text-transform: uppercase;
        margin-bottom: 0.4rem;
        display: block;
        text-shadow: 0 2px 4px rgba(0,0,0,0.5);
    }
    .message p {
        margin: 0;
        font-size: 1.05rem;
        opacity: 0.95;
    }
    .target {
        font-family: 'Fira Code', monospace;
        font-size: 0.9rem !important;
        background: rgba(0,0,0,0.3);
        padding: 0.3rem 0.6rem;
        border-radius: 6px;
        margin-top: 0.6rem !important;
        display: inline-block;
        border: 1px solid rgba(255,255,255,0.2);
    }
    .dismiss {
        background: none;
        border: none;
        color: white;
        font-size: 1.8rem;
        cursor: pointer;
        opacity: 0.7;
        padding: 0.5rem;
        transition: opacity 0.2s;
    }
    .dismiss:hover {
        opacity: 1;
    }
    
    @keyframes pulseFade {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.4; }
    }
</style>
