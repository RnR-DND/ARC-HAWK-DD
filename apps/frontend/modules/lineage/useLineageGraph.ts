import { useEffect, useState } from 'react';
import { fetchLineage, LineageResponse } from '../../services/lineage.api';
import { LineageNode, LineageEdge } from './lineage.types';

export interface UseLineageGraphReturn {
    nodes: LineageNode[];
    edges: LineageEdge[];
    aggregations: LineageResponse['aggregations'] | null;
    loading: boolean;
    error: string | null;
    refetch: () => void;
}

export function useLineageGraph(
    systemFilter?: string,
    riskFilter?: string
): UseLineageGraphReturn {
    const [nodes, setNodes] = useState<LineageNode[]>([]);
    const [edges, setEdges] = useState<LineageEdge[]>([]);
    const [aggregations, setAggregations] = useState<LineageResponse['aggregations'] | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchData = async () => {
        try {
            setLoading(true);
            setError(null);

            const data = await fetchLineage(systemFilter, riskFilter);

            setNodes(data.hierarchy.nodes);
            setEdges(data.hierarchy.edges);
            setAggregations(data.aggregations);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to fetch lineage');
            console.error('Lineage fetch error:', err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
    }, [systemFilter, riskFilter]);

    return {
        nodes,
        edges,
        aggregations,
        loading,
        error,
        refetch: fetchData,
    };
}
