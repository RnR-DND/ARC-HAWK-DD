/**
 * ARC-Hawk Design System - Themes (Production)
 * 
 * Centralized export of all design tokens
 */

import { colors, getNodeColor, getEdgeColor } from './colors';
import typography from './typography';
import layout from './spacing';

// ============================================
// THEME OBJECT
// ============================================

export const theme = {
    colors: colors.semantic,
    background: colors.background,
    border: colors.border,
    text: colors.text,
    nodeColors: colors.nodeColors,
    state: colors.state,
    typography: typography.textStyles,
    font: typography.fontFamily,
    fontSize: typography.fontSize,
    fontWeight: typography.fontWeight,
    lineHeight: typography.lineHeight,
    letterSpacing: typography.letterSpacing,
    spacing: layout.spacing,
    semanticSpacing: layout.semanticSpacing,
    borderRadius: layout.borderRadius,
    shadows: layout.shadows,
    zIndex: layout.zIndex,
} as const;

// ============================================
// EXPORTS
// ============================================

export default theme;
export { colors, typography, layout, getNodeColor, getEdgeColor };
