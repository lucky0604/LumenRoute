// LumenRoute Design System — Token Definitions
// Authority: agent_flow/thoughts/20260509-lumenroute-design-system.md
// Layer 1: Reference tokens → Layer 2: System tokens → Layer 3: Component tokens

// ── Dark Theme Reference Tokens ──
const darkColors = {
  canvas: "#0B1220",
  shell: "#0F172A",
  surface: "#111827",
  surfaceRaised: "#172033",
  borderSubtle: "#243041",
  borderStrong: "#334155",
  textPrimary: "#E5EEF9",
  textSecondary: "#94A3B8",
  textTertiary: "#64748B",
  accentPrimary: "#60A5FA",
  accentPrimaryStrong: "#3B82F6",
} as const;

// ── Light Theme Reference Tokens ──
const lightColors = {
  canvas: "#F4F7FB",
  shell: "#FFFFFF",
  surface: "#FFFFFF",
  surfaceRaised: "#F8FAFC",
  borderSubtle: "#E2E8F0",
  borderStrong: "#CBD5E1",
  textPrimary: "#0F172A",
  textSecondary: "#475569",
  textTertiary: "#64748B",
  accentPrimary: "#2563EB",
  accentPrimaryStrong: "#1D4ED8",
} as const;

// ── Semantic Status Tokens (shared between themes) ──
export const statusTokens = {
  success: { dark: "#22C55E", light: "#16A34A" },
  warning: { dark: "#F59E0B", light: "#D97706" },
  danger:  { dark: "#F43F5E", light: "#DC2626" },
  info:    { dark: "#38BDF8", light: "#0284C7" },
  neutral: { dark: "#94A3B8", light: "#64748B" },
} as const;

// ── Theme mode type ──
export type ThemeMode = "dark" | "light";

// ── Typography Tokens ──
export const typographyTokens = {
  fontSans: `"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,
  fontMono: `"IBM Plex Mono", "JetBrains Mono", "SFMono-Regular", Consolas, monospace`,
  fontSizeDisplay: 28,
  fontSizePageTitle: 24,
  fontSizeSectionTitle: 18,
  fontSizeBody: 14,
  fontSizeTable: 13,
  fontSizeMeta: 12,
  fontSizeCode: 13,
} as const;

// ── Spacing Tokens ──
export const spacingTokens = {
  pagePaddingDesktop: 24,
  pagePaddingTablet: 16,
  pagePaddingMobile: 12,
  toolbarGap: 8,
  panelPaddingDefault: 16,
  panelPaddingDense: 12,
  scale: [4, 8, 12, 16, 20, 24, 32, 40, 48] as const,
} as const;

// ── Radius Tokens ──
export const radiusTokens = {
  shell: 16,
  panel: 12,
  control: 10,
  badge: 999,
} as const;

// ── Motion Tokens ──
export const motionTokens = {
  transitionCommon: "150ms",
  transitionMajor: "300ms",
} as const;

// ── Build Ant Design theme config for a given mode ──
export function buildThemeConfig(mode: ThemeMode) {
  const colors = mode === "dark" ? darkColors : lightColors;
  const st = (key: keyof typeof statusTokens) => statusTokens[key][mode];

  return {
    token: {
      // ── Color tokens mapped to Ant Design ──
      colorPrimary: colors.accentPrimary,
      colorPrimaryHover: colors.accentPrimaryStrong,
      colorSuccess: st("success"),
      colorWarning: st("warning"),
      colorError: st("danger"),
      colorInfo: st("info"),

      colorBgLayout: colors.canvas,
      colorBgContainer: colors.surface,
      colorBgElevated: colors.surfaceRaised,
      colorBgSpotlight: colors.surfaceRaised,

      colorBorder: colors.borderSubtle,
      colorBorderSecondary: colors.borderSubtle,

      colorText: colors.textPrimary,
      colorTextSecondary: colors.textSecondary,
      colorTextTertiary: colors.textTertiary,

      // ── Typography ──
      fontFamily: typographyTokens.fontSans,
      fontSize: typographyTokens.fontSizeBody,

      // ── Shape ──
      borderRadius: radiusTokens.control,
      borderRadiusLG: radiusTokens.panel,
      borderRadiusSM: radiusTokens.control,
    },

    components: {
      Layout: {
        bodyBg: colors.canvas,
        headerBg: colors.shell,
        siderBg: colors.shell,
        triggerBg: colors.shell,
        triggerColor: colors.textSecondary,
      },
      Menu: {
        darkItemBg: colors.shell,
        darkSubMenuItemBg: colors.shell,
        darkItemColor: colors.textSecondary,
        darkItemSelectedBg: colors.surfaceRaised,
        darkItemSelectedColor: colors.accentPrimary,
        itemBg: colors.shell,
        subMenuItemBg: colors.shell,
        itemColor: colors.textSecondary,
        itemSelectedBg: colors.surfaceRaised,
        itemSelectedColor: colors.accentPrimary,
        groupTitleColor: colors.textTertiary,
        groupTitleFontSize: 11,
      },
      Table: {
        headerBg: colors.surface,
        headerColor: colors.textSecondary,
        rowHoverBg: colors.surfaceRaised,
        borderColor: colors.borderSubtle,
        cellFontSize: typographyTokens.fontSizeTable,
      },
      Tag: {
        defaultBg: colors.surfaceRaised,
        defaultColor: colors.textSecondary,
      },
      Badge: {
        colorText: colors.textPrimary,
      },
      Alert: {
        colorErrorBg: mode === "dark" ? "#3B1120" : "#FEF2F2",
        colorErrorBorder: mode === "dark" ? "#7F1D1D" : "#FECACA",
        colorWarningBg: mode === "dark" ? "#2D1A08" : "#FFFBEB",
        colorWarningBorder: mode === "dark" ? "#78350F" : "#FED7AA",
        colorInfoBg: mode === "dark" ? "#0C2D48" : "#EFF6FF",
        colorInfoBorder: mode === "dark" ? "#075985" : "#BFDBFE",
        colorSuccessBg: mode === "dark" ? "#052E16" : "#F0FDF4",
        colorSuccessBorder: mode === "dark" ? "#14532D" : "#BBF7D0",
      },
      Card: {
        headerBg: colors.surface,
      },
      Input: {
        activeBorderColor: colors.accentPrimary,
        hoverBorderColor: colors.borderStrong,
      },
      Select: {
        optionSelectedBg: colors.surfaceRaised,
      },
    },
  } as const;
}

// ── Export raw color references for custom component use ──
export { darkColors, lightColors };
