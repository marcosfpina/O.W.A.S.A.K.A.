<script lang="ts">
    import { onMount } from 'svelte';
    import { networkEvents } from '$lib/websocket';
    import * as d3 from 'd3';

    let svgContainer: HTMLElement;

    let nodes: any[] = [];
    let links: any[] = [];
    let simulation: d3.Simulation<any, any>;

    const NODE_COLORS: Record<string, string> = {
        host:      '#00ffd5',
        router:    '#ffd700',
        container: '#7b68ee',
        vm:        '#ff8c00',
        unknown:   '#888888',
        THREAT:    '#ff3333',
    };

    onMount(async () => {
        const width = svgContainer.clientWidth;
        const height = svgContainer.clientHeight || 450;

        const svg = d3.select(svgContainer)
            .append("svg")
            .attr("width", "100%")
            .attr("height", "100%")
            .attr("viewBox", `0 0 ${width} ${height}`);

        const g = svg.append("g");

        // Force simulation initialization
        simulation = d3.forceSimulation()
            .force("link", d3.forceLink().id((d: any) => d.id).distance(120))
            .force("charge", d3.forceManyBody().strength(-250))
            .force("center", d3.forceCenter(width / 2, height / 2))
            .force("collide", d3.forceCollide().radius(40));

        let linkSelection = g.append("g").attr("class", "links").selectAll(".link");
        let nodeSelection = g.append("g").attr("class", "nodes").selectAll(".node");

        // Load initial topology snapshot from REST
        try {
            const resp = await fetch('http://127.0.0.1:8080/api/topology');
            if (resp.ok) {
                const snap = await resp.json();
                nodes = (snap.nodes || []).map((n: any) => ({...n}));
                links = (snap.links || []).map((l: any) => ({...l}));
                updateGraph();
            }
        } catch (_) { /* backend not yet ready — wait for WS events */ }

        function updateGraph() {
            // Re-bind links
            linkSelection = linkSelection.data(links, d => d.source.id + "-" + d.target.id);
            linkSelection.exit().remove();
            const linkEnter = linkSelection.enter().append("line")
                .attr("class", "link")
                .style("stroke", "rgba(0, 255, 255, 0.2)")
                .style("stroke-width", 1.5);
            linkSelection = linkEnter.merge(linkSelection as any);

            // Re-bind nodes
            nodeSelection = nodeSelection.data(nodes, d => d.id);
            nodeSelection.exit().remove();
            
            const nodeEnter = nodeSelection.enter().append("g")
                .attr("class", "node")
                .style("cursor", "grab")
                .call(d3.drag()
                    .on("start", dragstarted)
                    .on("drag", dragged)
                    .on("end", dragended) as any);

            nodeEnter.append("circle")
                .attr("r", 10)
                .attr("fill", (d: any) => NODE_COLORS[d.type] ?? '#00ffd5')
                .attr("stroke", "rgba(255,255,255,0.2)")
                .attr("stroke-width", 2);

            nodeEnter.append("text")
                .attr("dx", 15)
                .attr("dy", ".35em")
                .text(d => d.id)
                .style("fill", "#e0e0e0")
                .style("font-size", "11px")
                .style("font-family", "monospace");

            nodeSelection = nodeEnter.merge(nodeSelection as any);

            // Restart physics
            simulation.nodes(nodes);
            (simulation.force("link") as any).links(links);
            simulation.alpha(1).restart();
        }

        simulation.on("tick", () => {
            linkSelection
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);

            nodeSelection.attr("transform", d => `translate(${d.x},${d.y})`);
        });

        // Live stream bindings
        const unsubscribe = networkEvents.subscribe(events => {
            if (!events.length) return;
            const ev = events[0];

            // Full topology replacement from backend mapper
            if (ev.type === 'TOPOLOGY_UPDATE' && ev.data) {
                nodes = (ev.data.nodes || []).map((n: any) => ({...n}));
                links = (ev.data.links || []).map((l: any) => ({...l}));
                updateGraph();
                return;
            }

            // Incremental update from raw network events
            let sourceId = ev.source || "Unknown";
            let destId = ev.destination || "Broadcast";

            if (ev.type === 'THREAT_ALERT') destId = ev.metadata?.target || destId;

            const nodeType = ev.type === 'THREAT_ALERT' ? 'THREAT' : 'unknown';

            let sNode = nodes.find((n: any) => n.id === sourceId);
            if (!sNode) {
                sNode = { id: sourceId, label: sourceId, type: nodeType };
                nodes.push(sNode);
            }
            if (ev.type === 'THREAT_ALERT') sNode.type = 'THREAT';

            let dNode = nodes.find((n: any) => n.id === destId);
            if (!dNode) {
                dNode = { id: destId, label: destId, type: nodeType };
                nodes.push(dNode);
            }
            if (ev.type === 'THREAT_ALERT') dNode.type = 'THREAT';

            const existingLink = links.find((l: any) =>
                (l.source?.id === sourceId || l.source === sourceId) &&
                (l.target?.id === destId   || l.target === destId)
            );
            if (!existingLink && sourceId !== destId) {
                links.push({ source: sourceId, target: destId });
            }

            if (nodes.length > 40) {
                nodes = nodes.slice(-40);
                const validIds = new Set(nodes.map((n: any) => n.id));
                links = links.filter((l: any) =>
                    validIds.has(l.source?.id ?? l.source) &&
                    validIds.has(l.target?.id ?? l.target)
                );
            }

            updateGraph();
        });

        // D3 Drag Event Handlers
        function dragstarted(event: any, d: any) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
            d3.select(this).style("cursor", "grabbing");
        }

        function dragged(event: any, d: any) {
            d.fx = event.x;
            d.fy = event.y;
        }

        function dragended(event: any, d: any) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
            d3.select(this).style("cursor", "grab");
        }

        return () => {
            unsubscribe();
            simulation.stop();
        };
    });
</script>

<div class="topology-container" bind:this={svgContainer}>
</div>

<style>
    .topology-container {
        width: 100%;
        height: 100%;
        min-height: 450px;
        position: relative;
        background: rgba(0,0,0,0.1);
        border: 1px solid rgba(255,255,255,0.05);
        border-radius: 8px;
        overflow: hidden;
    }
</style>
