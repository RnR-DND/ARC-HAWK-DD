import { Node, Edge, Position } from 'reactflow';
import dagre from 'dagre';
import { DEFAULT_LAYOUT_CONFIG } from './lineage.types';

/**
 * Helper to get connected nodes (BFS)
 */
export const getConnectedNodes = (startNodeId: string, edges: Edge[], maxDepth: number = 2): Set<string> => {
    const connected = new Set<string>();
    connected.add(startNodeId);

    let currentLevel = new Set<string>([startNodeId]);

    for (let i = 0; i < maxDepth; i++) {
        const nextLevel = new Set<string>();
        edges.forEach(edge => {
            if (currentLevel.has(edge.source)) {
                connected.add(edge.target);
                nextLevel.add(edge.target);
            }
            if (currentLevel.has(edge.target)) {
                connected.add(edge.source);
                nextLevel.add(edge.source);
            }
        });
        currentLevel = nextLevel;
    }
    return connected;
};

/**
 * Apply Dagre hierarchical layout to nodes
 */
export function getLayoutedElements(nodes: Node[], edges: Edge[]) {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));

    dagreGraph.setGraph({
        rankdir: DEFAULT_LAYOUT_CONFIG.rankdir,
        nodesep: DEFAULT_LAYOUT_CONFIG.nodesep,
        ranksep: DEFAULT_LAYOUT_CONFIG.ranksep,
        edgesep: DEFAULT_LAYOUT_CONFIG.edgesep,
        marginx: DEFAULT_LAYOUT_CONFIG.marginx,
        marginy: DEFAULT_LAYOUT_CONFIG.marginy,
    });

    nodes.forEach((node) => {
        // Dynamic sizing matching LineageNode.tsx
        let width = 240;
        let height = 100;

        // Ensure these match LineageNode.tsx exactly or are slightly larger
        switch (node.data.type) {
            case 'system':
                width = 320; // Increased width
                height = 180; // Increased to safe height for text wrapping
                break;
            case 'asset':
            case 'file':
            case 'table':
                width = 280;
                height = 160; // Increased height
                break;
            case 'data_category':
            case 'category':
                width = 240;
                height = 140; // Increased height
                break;
            case 'finding':
                width = 220;
                height = 120; // Increased height
                break;
            default:
                width = 240;
                height = 140;
        }

        dagreGraph.setNode(node.id, { width, height });
    });

    edges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = nodes.map((node) => {
        const nodeWithPosition = dagreGraph.node(node.id);

        if (!nodeWithPosition) return node;

        // Recalculate dimensions for centering (must match above)
        let width = 240;
        let height = 100;
        switch (node.data.type) {
            case 'system': width = 320; height = 180; break;
            case 'asset': case 'file': case 'table': width = 280; height = 160; break;
            case 'data_category': case 'category': width = 240; height = 140; break;
            case 'finding': width = 220; height = 120; break;
            default: width = 240; height = 140;
        }

        return {
            ...node,
            targetPosition: Position.Left,
            sourcePosition: Position.Right,
            position: {
                x: nodeWithPosition.x - width / 2,
                y: nodeWithPosition.y - height / 2,
            },
        };
    });

    return { nodes: layoutedNodes, edges };
}
