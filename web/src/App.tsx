import { useState, useCallback, useEffect } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { ConfigProvider, App as AntApp } from "antd";
import { buildThemeConfig } from "./theme/tokens";
import type { ThemeMode } from "./theme/tokens";
import LoginPage from "./pages/LoginPage";
import ProvidersPage from "./pages/ProvidersPage";
import RoutesPage from "./pages/RoutesPage";
import ApiKeysPage from "./pages/ApiKeysPage";
import RequestLogsPage from "./pages/RequestLogsPage";
import HealthSummaryPage from "./pages/HealthSummaryPage";
import ModelPerformancePage from "./pages/ModelPerformancePage";
import TargetDiagnosisPage from "./pages/TargetDiagnosisPage";
import ControlCenterPage from "./pages/ControlCenterPage";
import AdminLayout from "./components/AdminLayout";

const THEME_STORAGE_KEY = "lumenroute-theme";

function getInitialTheme(): ThemeMode {
  const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === "dark" || stored === "light") return stored;
  return "dark";
}

function App() {
  const [themeMode, setThemeMode] = useState<ThemeMode>(getInitialTheme);
  const themeConfig = buildThemeConfig(themeMode);

  const toggleTheme = useCallback(() => {
    setThemeMode((prev) => {
      const next = prev === "dark" ? "light" : "dark";
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
      return next;
    });
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", themeMode);
  }, [themeMode]);

  return (
    <ConfigProvider theme={themeConfig}>
      <AntApp>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route element={<AdminLayout themeMode={themeMode} onToggleTheme={toggleTheme} />}>
            <Route path="/control-center" element={<ControlCenterPage />} />
            <Route path="/providers" element={<ProvidersPage />} />
            <Route path="/routes" element={<RoutesPage />} />
            <Route path="/api-keys" element={<ApiKeysPage />} />
            <Route path="/request-logs" element={<RequestLogsPage />} />
            <Route path="/model-performance" element={<ModelPerformancePage />} />
            <Route path="/diagnostics/targets/:id" element={<TargetDiagnosisPage />} />
            <Route path="/health" element={<HealthSummaryPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </AntApp>
    </ConfigProvider>
  );
}

export default App;
