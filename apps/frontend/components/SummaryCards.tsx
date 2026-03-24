import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';

interface SummaryCardsProps {
    totalFindings?: number;
    sensitivePIICount?: number;
    highRiskAssets?: number;
    criticalFindings?: number;
}

export default function SummaryCards({
    totalPII = 0,
    highRiskFindings = 0,
    assetsHit = 0,
    actionsRequired = 0,
}: {
    totalPII?: number;
    highRiskFindings?: number;
    assetsHit?: number;
    actionsRequired?: number;
}) {
    return (
        <div
            style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
                gap: '24px',
                marginBottom: '32px',
            }}
        >
            <Card
                label="Total PII"
                value={totalPII}
                subtitle="Detections across all sources"
                color={colors.text.primary}
                tooltip="Total count of PII instances detected."
            />

            <Card
                label="High Risk"
                value={highRiskFindings}
                subtitle="High confidence or sensitive"
                color={colors.state.risk}
                tooltip="Findings classified as High Risk."
            />

            <Card
                label="Assets Hit"
                value={assetsHit}
                subtitle="Containing PII"
                color={colors.nodeColors.asset}
                tooltip="Number of assets (tables, files, buckets) containing PII."
            />

            <Card
                label="Actions Req"
                value={actionsRequired}
                subtitle="Pending review or remediation"
                color={colors.state.warning}
                tooltip="Findings requiring confirmation or remediation."
            />
        </div>
    );
}

function Card({
    label,
    value,
    subtitle,
    color,
    tooltip,
}: {
    label: string;
    value: number;
    subtitle: string;
    color: string;
    tooltip?: string;
}) {
    const [isHovered, setIsHovered] = React.useState(false);

    return (
        <div
            title={tooltip} // Native tooltip for simplicity
            style={{
                background: colors.background.surface,
                border: `1px solid ${colors.border.default}`,
                borderRadius: theme.borderRadius.xl,
                padding: '24px',
                boxShadow: isHovered ? theme.shadows.lg : theme.shadows.sm,
                transition: 'all 0.2s ease',
                transform: isHovered ? 'translateY(-4px)' : 'translateY(0)',
                cursor: 'help', // Cursor changed to indicate interactivity
            }}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >


            <div
                style={{
                    fontSize: '11px',
                    color: colors.text.secondary,
                    marginBottom: '10px',
                    fontWeight: 800,
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                }}
            >
                {label}
            </div>
            <div
                style={{
                    fontSize: '42px',
                    fontWeight: 900,
                    color: color,
                    lineHeight: 1,
                    marginBottom: '8px',
                    letterSpacing: '-0.03em',
                }}
            >
                {value.toLocaleString()}
            </div>
            <div
                style={{
                    fontSize: '13px',
                    color: colors.text.muted,
                    fontWeight: 600,
                }}
            >
                {subtitle}
            </div>
        </div>
    );
}
