import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#070A0F",
          900: "#0B0F16",
          850: "#0D131C",
          800: "#111722",
          700: "#1A2230",
        },
        line: "#1C2530",
        signal: {
          DEFAULT: "#34D399",
          dim: "#10B981",
          soft: "#6EE7B7",
        },
        severity: {
          critical: "#F87171",
          high: "#FB923C",
          medium: "#FBBF24",
          low: "#60A5FA",
          informational: "#94A3B8",
        },
      },
      fontFamily: {
        sans: ["var(--font-inter)", "system-ui", "sans-serif"],
        mono: ["var(--font-mono)", "ui-monospace", "SFMono-Regular", "monospace"],
      },
      boxShadow: {
        glow: "0 0 0 1px rgba(52, 211, 153, 0.25), 0 18px 60px -20px rgba(52, 211, 153, 0.35)",
      },
      backgroundImage: {
        "grid-fade":
          "linear-gradient(to bottom, rgba(7,10,15,0.2), rgba(7,10,15,1)), radial-gradient(circle at 1px 1px, rgba(148,163,184,0.14) 1px, transparent 0)",
      },
      backgroundSize: {
        grid: "100% 100%, 32px 32px",
      },
    },
  },
  plugins: [],
};

export default config;