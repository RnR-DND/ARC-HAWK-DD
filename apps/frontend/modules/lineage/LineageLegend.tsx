import React from 'react';
import { colors } from '@/design-system/colors';

export default function LineageLegend() {
    return (
        <div
            style={{
                position: 'absolute',
                top: '16px',
                right: '16px',
                background: colors.background.surface,
                backdropFilter: 'blur(10px)',
                padding: '12px 16px',
                borderRadius: '8px',
                border: `1px solid ${colors.border.default}`,
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
                fontSize: '12px',
                color: colors.text.secondary,
                display: 'flex',
                flexDirection: 'column',
                gap: '8px',
                zIndex: 10
            }}
        >
            <div style={{ fontWeight: 600, color: colors.text.primary, marginBottom: '4px' }}>Legend</div>
            <LegendItem color={colors.nodeColors.system} label="System" />
            <LegendItem color={colors.nodeColors.asset} label="Asset" />
            <LegendItem color={colors.nodeColors.category} label="Category" />
            <LegendItem color={colors.state.risk} label="Critical" />
        </div>
    );
}

function LegendItem({ color, label }: { color: string; label: string }) {
    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <div
                style={{
                    width: '12px',
                    height: '12px',
                    borderRadius: '3px',
                    backgroundColor: color,
                }}
            />
            <span>{label}</span>
        </div>
    );
}
