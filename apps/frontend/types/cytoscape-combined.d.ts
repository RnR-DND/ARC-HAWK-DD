import cytoscape from 'cytoscape';

declare module 'cytoscape-dagre';

declare module 'cytoscape' {
    interface BaseLayoutOptions {
        // Dagre layout options
        rankDir?: string;
        rankdir?: string;
        align?: string;
        nodesep?: number;
        edgesep?: number;
        ranksep?: number;
        marginx?: number;
        marginy?: number;
    }
}
